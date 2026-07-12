/**
 * BackgroundPicker.tsx — 课件背景图库选择器（批次2新建，批次3扩展生产入口 v2）
 *
 * 嵌入位置：课件工坊 Step3「确认导航栏」页 + Step5「确认提交」页（小修8）。
 * 交互：
 *   - 画廊展示全部可用图集（系统在前、我的在后），点选秒换全部已生成页（零token）。
 *   - 「✨AI生成一套」：按课件主题/学科/年级预填提示词（可编辑），调豆包出封面+内页两张
 *     16:9图→上OSS→存个人集→自动应用到本课件。
 *   - 「📤上传一套」：本地两张图(≤5MB, JPG/PNG/WEBP)→OSS→个人集→自动应用。
 *   - 个人集卡片带「删除」（=归档，已选课件不受影响）；admin额外有「存为系统图库」。
 *   - 小修6：缩略图URL追加 ?x-oss-process=image/resize,w_400，不再加载原图。
 *   - 小修7：disabled=true（批量生成进行中）时整体禁用，防在飞页落库带旧背景的竞态。
 * 选中判定：后端只存URL快照不存集ID，故以 cover_bg_url === 集.cover_public_url 判定。
 */
import { useState, useEffect, useCallback } from 'react'
import {
  listBackgroundSets, getCWBackground, setCWBackground, clearCWBackground,
  generateBackgroundSet, uploadBackgroundSet, deleteBackgroundSet, promoteBackgroundSet,
  CW_STYLE_CONFIG,
  getPageBackground, setPageBackground, clearPageBackground,
} from '@/api/coursewares'
import type { PageBgSetting } from '@/api/coursewares'
import type { CWBackgroundSet } from '@/api/coursewares'
import { useAuth } from '@/store/auth'
import { C } from './workshopConstants'

interface Props {
  coursewareId: string
  /** 选/清背景成功后回调（父组件重拉课件与页面，刷新预览） */
  onSwapped: () => void
  /** 小修7：批量生成进行中传true，整体禁用防竞态 */
  disabled?: boolean
  /** AI生成提示词预填上下文（课件标题/学科/年级），可不传 */
  cwTitle?: string
  cwSubject?: string
  cwGrade?: string
  /** 当前选中的页码（传入后显示"本页背景"区块，可逐页设蒙版/换背景） */
  pageNum?: number
}

/** 小修6：OSS缩略图——追加阿里云图片处理参数，缩略到宽400不再拉原图 */
const thumb = (url: string) => {
  if (!url) return url
  if (url.includes('aliyuncs.com') && !url.includes('?')) return url + '?x-oss-process=image/resize,w_400'
  return url
}

// 批次3c：风格英文值→中文标签（与模板画廊共用同一词典，未收录的值兜底显示原文）
const STYLE_LABELS: Record<string, string> = {
  minimalist: '简约清新', playful: '活泼趣味', tech: '科技感',
  academic: '学术严谨', organic: '自然雅致', immersive: '3D沉浸式',
}

export default function BackgroundPicker({ coursewareId, onSwapped, disabled = false, cwTitle = '', cwSubject = '', cwGrade = '', pageNum }: Props) {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const [sets, setSets] = useState<CWBackgroundSet[]>([])
  const [coverBg, setCoverBg] = useState('')      // 当前生效的头图URL（''=未选背景）
  const [loading, setLoading] = useState(true)
  const [applying, setApplying] = useState('')    // 正在应用的集ID（'__clear__'=清除中, ''=空闲）
  const [message, setMessage] = useState('')

  // 批次3：生产入口面板状态（''=收起 / 'gen'=AI生成 / 'upload'=上传）
  const [panel, setPanel] = useState<'' | 'gen' | 'upload'>('')
  const [genName, setGenName] = useState('')
  const [genCover, setGenCover] = useState('')
  const [genContent, setGenContent] = useState('')
  const [genRunning, setGenRunning] = useState(false)
  const [upName, setUpName] = useState('')
  const [upCover, setUpCover] = useState<File | null>(null)
  const [upContent, setUpContent] = useState<File | null>(null)
  const [upRunning, setUpRunning] = useState(false)
  const [deletingId, setDeletingId] = useState('')
  const [promotingId, setPromotingId] = useState('')
  // 批次3c：风格筛选（''=全部），选项由当前图集数据动态收集
  const [styleFilter, setStyleFilter] = useState('')

  // 任一生产/应用动作进行中 → 其它入口互斥
  const busy = !!applying || genRunning || upRunning || !!deletingId || !!promotingId

  // 重拉图集列表 + 课件当前选择
  const reload = useCallback(async () => {
    const [ls, sel] = await Promise.all([listBackgroundSets(), getCWBackground(coursewareId)])
    setSets(ls.sets || [])
    setCoverBg(sel.cover_bg_url || '')
  }, [coursewareId])

  // 进入即拉取（防卸载后 setState 用 cancelled 标志）
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    Promise.all([listBackgroundSets(), getCWBackground(coursewareId)])
      .then(([ls, sel]) => {
        if (cancelled) return
        setSets(ls.sets || [])
        setCoverBg(sel.cover_bg_url || '')
      })
      .catch(() => { if (!cancelled) setMessage('❌ 背景图库加载失败，可刷新页面重试') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [coursewareId])

  // 应用某集（set=null 表示清除）
  const apply = async (set: CWBackgroundSet | null) => {
    if (busy || disabled) return
    setApplying(set ? set.id : '__clear__')
    setMessage(set ? '⏳ 正在应用背景并秒换已生成页…' : '⏳ 正在清除背景…')
    try {
      const res = set ? await setCWBackground(coursewareId, set.id) : await clearCWBackground(coursewareId)
      setCoverBg(res.cover_bg_url || '')
      if (set) {
        setMessage('✅ 已应用「' + set.name + '」' +
          (res.swapped_pages > 0 ? '，已秒换 ' + res.swapped_pages + ' 个已生成页' : '') +
          '，后续生成的页面将自动使用该背景')
      } else {
        setMessage('✅ 已清除背景' + (res.swapped_pages > 0 ? '，已回退 ' + res.swapped_pages + ' 个已生成页' : ''))
      }
      onSwapped()
    } catch (e) {
      setMessage('❌ 操作失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setApplying('') }
  }

  // 打开AI生成面板：首次按课件上下文预填提示词（可编辑）
  const openGenPanel = () => {
    if (busy || disabled) return
    if (!genCover) {
      const subjGrade = [cwSubject, cwGrade].filter(Boolean).join('、')
      setGenCover('「' + (cwTitle || '本课件') + '」课件封面背景图：' + (subjGrade ? '符合' + subjGrade + '学生气质的' : '') +
        '横版高清插画背景，画面主体偏右、左上方留白便于叠加标题文字，色彩协调有质感，画面不包含任何文字')
      setGenContent('与封面同一风格体系的内页底纹背景：同色系、更素雅简洁，仅保留轻微的装饰元素')
    }
    setPanel(panel === 'gen' ? '' : 'gen')
  }

  // AI生成一套（后端：豆包出两张16:9 → OSS → 个人集 → 自动应用本课件）
  const handleGenerate = async () => {
    if (busy || disabled || !genCover.trim() || !genContent.trim()) return
    setGenRunning(true)
    setMessage('🤖 AI正在生成封面+内页两张背景图（约1-3分钟），生成后将自动应用到本课件…')
    try {
      const res = await generateBackgroundSet({
        courseware_id: coursewareId,
        name: genName.trim() || undefined,
        cover_prompt: genCover.trim(),
        content_prompt: genContent.trim(),
      })
      await reload()
      if (res.selection) {
        setCoverBg(res.selection.cover_bg_url || '')
        setMessage('✅ 「' + res.set.name + '」已生成并应用' +
          (res.selection.swapped_pages > 0 ? '，已秒换 ' + res.selection.swapped_pages + ' 个已生成页' : ''))
        onSwapped()
      } else {
        setMessage('✅ 「' + res.set.name + '」已生成并存入我的图库，可在下方点选应用')
      }
      setPanel('')
    } catch (e) {
      setMessage('❌ AI生成失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setGenRunning(false) }
  }

  // 上传一套
  const handleUpload = async () => {
    if (busy || disabled || !upCover || !upContent) return
    setUpRunning(true)
    setMessage('⏳ 正在上传两张背景图到云盘…')
    try {
      const res = await uploadBackgroundSet({
        name: upName.trim() || undefined,
        coursewareId,
        cover: upCover,
        content: upContent,
      })
      await reload()
      if (res.selection) {
        setCoverBg(res.selection.cover_bg_url || '')
        setMessage('✅ 「' + res.set.name + '」已上传并应用' +
          (res.selection.swapped_pages > 0 ? '，已秒换 ' + res.selection.swapped_pages + ' 个已生成页' : ''))
        onSwapped()
      } else {
        setMessage('✅ 「' + res.set.name + '」已上传并存入我的图库，可在下方点选应用')
      }
      setPanel(''); setUpName(''); setUpCover(null); setUpContent(null)
    } catch (e) {
      setMessage('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setUpRunning(false) }
  }

  // 删除（归档）图集
  const handleDelete = async (set: CWBackgroundSet) => {
    if (busy || disabled) return
    if (!confirm('确定删除背景图集「' + set.name + '」？\n已选用它的课件不受影响（背景以快照保存），但图库中将不再显示。')) return
    setDeletingId(set.id)
    try {
      await deleteBackgroundSet(set.id)
      setSets(prev => prev.filter(s => s.id !== set.id))
      setMessage('✅ 已删除「' + set.name + '」')
    } catch (e) {
      setMessage('❌ 删除失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setDeletingId('') }
  }

  // admin：升级为系统图库
  const handlePromote = async (set: CWBackgroundSet) => {
    if (busy || disabled) return
    if (!confirm('将「' + set.name + '」存为系统图库？升级后全体用户可见、可使用。')) return
    setPromotingId(set.id)
    try {
      await promoteBackgroundSet(set.id)
      await reload()
      setMessage('✅ 「' + set.name + '」已存为系统图库')
    } catch (e) {
      setMessage('❌ 升级失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setPromotingId('') }
  }

  // 文件选择（cover/content 共用）
  const pickFile = (slot: 'cover' | 'content') => {
    const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/jpeg,image/png,image/webp'
    inp.onchange = (ev) => {
      const f = (ev.target as HTMLInputElement).files?.[0]
      if (!f) return
      if (f.size > 5 * 1024 * 1024) { setMessage('❌ 图片不能超过5MB'); return }
      if (slot === 'cover') setUpCover(f); else setUpContent(f)
    }
    inp.click()
  }

  // 批次3c：从当前图集数据收集出现过的风格值
  const styleOptions = Array.from(new Set(sets.map(s => s.style_category).filter(Boolean)))
  const noneSelected = coverBg === ''
  const smallBtn = (active: boolean): React.CSSProperties => ({
    padding: '6px 14px', borderRadius: 8, fontSize: 12, fontWeight: 600,
    cursor: (busy || disabled) ? 'default' : 'pointer',
    border: '1px dashed ' + (active ? '#7C3AED' : C.border),
    background: active ? 'rgba(124,58,237,0.06)' : '#fff',
    color: active ? '#7C3AED' : C.textSecondary,
  })
  const inputStyle: React.CSSProperties = { padding: '8px 12px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 13, outline: 'none' }

  return (
    <div style={{ marginTop: 16, padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#FAFAFA', opacity: disabled ? 0.75 : 1 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 8, marginBottom: 4 }}>
        <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>🖼️ 课件背景图（可选）</div>
        {/* 批次3：生产入口（生成中/禁用态不可用） */}
        <div style={{ display: 'flex', gap: 8 }}>
          <button onClick={openGenPanel} disabled={busy || disabled} style={smallBtn(panel === 'gen')}>✨ AI生成一套</button>
          <button onClick={() => { if (!busy && !disabled) setPanel(panel === 'upload' ? '' : 'upload') }} disabled={busy || disabled} style={smallBtn(panel === 'upload')}>📤 上传一套</button>
        </div>
      </div>
      <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 12 }}>
        一组背景 = 封面头图 + 内页底纹。点选后已生成页立即换上新背景（无需重新生成），后续生成的页面自动使用。
      </div>

      {/* 小修7：批量生成进行中禁用提示 */}
      {disabled && (
        <div style={{ marginBottom: 12, padding: '10px 14px', borderRadius: 8, background: '#FEF3C7', color: '#92400E', fontSize: 13 }}>
          ⏳ 批量生成进行中，暂不能更换背景（避免正在生成的页面带上新旧混杂的背景）。生成完成后即可操作。
        </div>
      )}

      {/* 批次3：AI生成面板 */}
      {panel === 'gen' && !disabled && (
        <div style={{ marginBottom: 14, padding: 14, borderRadius: 10, border: '1px solid rgba(124,58,237,0.35)', background: 'rgba(124,58,237,0.03)' }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: '#7C3AED', marginBottom: 8 }}>✨ AI生成一套背景（已按本课件预填，可修改）</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <input value={genName} onChange={e => setGenName(e.target.value)} placeholder="图集名称（可留空，自动命名）" disabled={genRunning} style={inputStyle} />
            <textarea value={genCover} onChange={e => setGenCover(e.target.value)} rows={3} disabled={genRunning}
              placeholder="封面背景图提示词" style={{ ...inputStyle, resize: 'vertical' }} />
            <textarea value={genContent} onChange={e => setGenContent(e.target.value)} rows={2} disabled={genRunning}
              placeholder="内页底纹提示词（系统会自动追加：浅色低对比、适合做底纹）" style={{ ...inputStyle, resize: 'vertical' }} />
            <div style={{ fontSize: 11, color: C.textMuted }}>💡 两张图均为16:9高清(2560×1440)；内页会自动加"浅色低对比、适合做底纹"约束保证文字可读；生成约1-3分钟，成功后自动应用到本课件。</div>
            <div>
              <button onClick={handleGenerate} disabled={genRunning || !genCover.trim() || !genContent.trim()}
                style={{ padding: '10px 24px', borderRadius: 8, border: 'none', background: (!genRunning && genCover.trim() && genContent.trim()) ? 'linear-gradient(135deg, #7C3AED, #6D28D9)' : '#E5E7EB', color: (!genRunning && genCover.trim() && genContent.trim()) ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (!genRunning && genCover.trim() && genContent.trim()) ? 'pointer' : 'default' }}>
                {genRunning ? '⏳ 生成中（约1-3分钟）…' : '🤖 生成并应用到本课件'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 批次3：上传面板 */}
      {panel === 'upload' && !disabled && (
        <div style={{ marginBottom: 14, padding: 14, borderRadius: 10, border: '1px solid rgba(8,145,178,0.35)', background: 'rgba(8,145,178,0.03)' }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: '#0891B2', marginBottom: 8 }}>📤 上传一套背景（封面+内页两张，≤5MB，JPG/PNG/WEBP）</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <input value={upName} onChange={e => setUpName(e.target.value)} placeholder="图集名称（可留空，自动命名）" disabled={upRunning} style={inputStyle} />
            <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
              <button onClick={() => pickFile('cover')} disabled={upRunning} style={{ ...smallBtn(!!upCover), borderStyle: 'solid' }}>
                {upCover ? '✅ 封面: ' + (upCover.name.length > 18 ? upCover.name.slice(0, 18) + '…' : upCover.name) : '选择封面头图'}
              </button>
              <button onClick={() => pickFile('content')} disabled={upRunning} style={{ ...smallBtn(!!upContent), borderStyle: 'solid' }}>
                {upContent ? '✅ 内页: ' + (upContent.name.length > 18 ? upContent.name.slice(0, 18) + '…' : upContent.name) : '选择内页底纹图'}
              </button>
            </div>
            <div style={{ fontSize: 11, color: C.textMuted }}>💡 建议16:9横图；内页图请选浅色低对比的图，否则正文文字可能不易读。上传成功后自动应用到本课件。</div>
            <div>
              <button onClick={handleUpload} disabled={upRunning || !upCover || !upContent}
                style={{ padding: '10px 24px', borderRadius: 8, border: 'none', background: (!upRunning && upCover && upContent) ? 'linear-gradient(135deg, #0891B2, #0E7490)' : '#E5E7EB', color: (!upRunning && upCover && upContent) ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (!upRunning && upCover && upContent) ? 'pointer' : 'default' }}>
                {upRunning ? '⏳ 上传中…' : '📤 上传并应用到本课件'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 批次3c：风格筛选chips（仅当存在两种以上风格时显示） */}
      {!loading && styleOptions.length > 1 && (
        <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 10 }}>
          {['', ...styleOptions].map(st => (
            <button key={st || '__all__'} onClick={() => setStyleFilter(st)}
              style={{ padding: '4px 12px', borderRadius: 999, fontSize: 12, cursor: 'pointer',
                border: '1px solid ' + (styleFilter === st ? C.primary : C.border),
                background: styleFilter === st ? C.primaryBg : '#fff',
                color: styleFilter === st ? C.primary : C.textSecondary,
                fontWeight: styleFilter === st ? 600 : 400 }}>
              {st === '' ? '全部' : (CW_STYLE_CONFIG[st] ? CW_STYLE_CONFIG[st].emoji + ' ' + CW_STYLE_CONFIG[st].label : (STYLE_LABELS[st] || st))}
            </button>
          ))}
        </div>
      )}

      {loading && <div style={{ padding: '14px 0', fontSize: 13, color: C.textMuted, textAlign: 'center' }}>⏳ 背景图库加载中...</div>}

      {!loading && (
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
          {/* 不使用背景 */}
          <div onClick={() => { if (!busy && !disabled && !noneSelected) apply(null) }}
            style={{
              width: 200, borderRadius: 10, overflow: 'hidden', cursor: (busy || disabled || noneSelected) ? 'default' : 'pointer',
              border: '2px ' + (noneSelected ? 'solid ' + C.primary : 'dashed ' + C.border),
              background: noneSelected ? C.primaryBg : '#fff', transition: 'all 200ms',
            }}>
            <div style={{ height: 64, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 26, background: '#F3F4F6' }}>🚫</div>
            <div style={{ padding: '8px 10px' }}>
              <div style={{ fontSize: 13, fontWeight: 600, color: noneSelected ? C.primary : C.textPrimary }}>
                不使用背景{noneSelected ? '（当前）' : ''}
              </div>
              <div style={{ fontSize: 11, color: C.textMuted, marginTop: 2 }}>
                {applying === '__clear__' ? '⏳ 清除中...' : '使用模板默认背景'}
              </div>
            </div>
          </div>

          {/* 图集卡片：系统在前、我的在后（后端已排序）；缩略图走OSS缩放（小修6） */}
          {sets.filter(s => !styleFilter || s.style_category === styleFilter).map(s => {
            const selected = !!coverBg && coverBg === s.cover_public_url
            const cardBusy = applying === s.id || deletingId === s.id || promotingId === s.id
            const mine = s.scope === 'personal'
            return (
              <div key={s.id}
                style={{
                  width: 200, borderRadius: 10, overflow: 'hidden',
                  border: '2px solid ' + (selected ? C.primary : C.border),
                  background: selected ? C.primaryBg : '#fff', transition: 'all 200ms',
                }}>
                {/* 头图/内页双缩略图并排（点击应用） */}
                <div onClick={() => { if (!busy && !disabled && !selected) apply(s) }}
                  style={{ display: 'flex', gap: 2, background: '#F3F4F6', cursor: (busy || disabled || selected) ? 'default' : 'pointer' }}>
                  <div style={{ flex: 1, position: 'relative' }}>
                    <img src={thumb(s.cover_public_url)} alt="头图" loading="lazy"
                      style={{ width: '100%', height: 64, objectFit: 'cover', display: 'block' }} />
                    <span style={{ position: 'absolute', left: 4, bottom: 4, padding: '1px 6px', borderRadius: 4, background: 'rgba(0,0,0,0.55)', color: '#fff', fontSize: 10 }}>头图</span>
                  </div>
                  <div style={{ flex: 1, position: 'relative' }}>
                    <img src={thumb(s.content_public_url)} alt="内页" loading="lazy"
                      style={{ width: '100%', height: 64, objectFit: 'cover', display: 'block' }} />
                    <span style={{ position: 'absolute', left: 4, bottom: 4, padding: '1px 6px', borderRadius: 4, background: 'rgba(0,0,0,0.55)', color: '#fff', fontSize: 10 }}>内页</span>
                  </div>
                </div>
                <div style={{ padding: '8px 10px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span onClick={() => { if (!busy && !disabled && !selected) apply(s) }}
                      style={{ flex: 1, fontSize: 13, fontWeight: 600, color: selected ? C.primary : C.textPrimary, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', cursor: (busy || disabled || selected) ? 'default' : 'pointer' }}>{s.name}</span>
                    <span style={{ flexShrink: 0, padding: '1px 6px', borderRadius: 4, fontSize: 10, fontWeight: 600, color: s.scope === 'system' ? '#2563EB' : '#B45309', background: s.scope === 'system' ? '#EFF6FF' : 'rgba(245,158,11,0.1)' }}>
                      {s.scope === 'system' ? '系统' : '我的'}
                    </span>
                  </div>
                  <div style={{ fontSize: 11, color: C.textMuted, marginTop: 2, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={s.description}>
                    {cardBusy ? '⏳ 处理中...' : selected ? '✅ 当前使用中' : (s.description || '点击应用')}
                  </div>
                  {/* 批次3：个人集删除 + admin存为系统图库 */}
                  {(mine || isAdmin) && (
                    <div style={{ display: 'flex', gap: 4, marginTop: 6 }}>
                      {mine && isAdmin && (
                        <button onClick={() => handlePromote(s)} disabled={busy || disabled}
                          title="存为系统图库：全体用户可见可用"
                          style={{ padding: '3px 8px', borderRadius: 5, border: '1px solid #2563EB', background: 'rgba(37,99,235,0.06)', color: '#2563EB', fontSize: 10, cursor: (busy || disabled) ? 'default' : 'pointer' }}>
                          {promotingId === s.id ? '⏳' : '⬆ 存为系统'}
                        </button>
                      )}
                      {(mine || isAdmin) && (
                        <button onClick={() => handleDelete(s)} disabled={busy || disabled}
                          title="删除（归档）此图集：已选用它的课件不受影响"
                          style={{ padding: '3px 8px', borderRadius: 5, border: '1px solid #EF4444', background: 'rgba(239,68,68,0.06)', color: '#EF4444', fontSize: 10, cursor: (busy || disabled) ? 'default' : 'pointer' }}>
                          {deletingId === s.id ? '⏳' : '🗑 删除'}
                        </button>
                      )}
                    </div>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}


      {/* ==================== 页级背景覆盖（蒙版开关 + 单页背景图上传） ==================== */}
      {pageNum != null && pageNum > 0 && (
        <PageBgSection coursewareId={coursewareId} pageNum={pageNum} disabled={disabled || busy}
          onSwapped={onSwapped} setMessage={setMessage} />
      )}

      {message && (
        <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, fontSize: 13, background: message.startsWith('❌') ? '#FEE2E2' : message.startsWith('✅') ? '#D1FAE5' : '#EFF6FF', color: message.startsWith('❌') ? '#DC2626' : message.startsWith('✅') ? '#059669' : '#2563EB' }}>{message}</div>
      )}
    </div>
  )
}

// ==================== 页级背景覆盖子组件 ====================

/** 蒙版模式选项 */
const BG_MODES = [
  { key: 'default', label: '🎨 跟随默认', desc: '使用课件级默认蒙版' },
  { key: 'custom', label: '🎛️ 自定义透明度', desc: '调节蒙版浓淡' },
  { key: 'none', label: '🖼️ 无蒙版', desc: '纯背景图不加蒙版' },
] as const

interface PageBgProps {
  coursewareId: string
  pageNum: number
  disabled: boolean
  onSwapped: () => void
  setMessage: (msg: string) => void
}

function PageBgSection({ coursewareId, pageNum, disabled, onSwapped, setMessage }: PageBgProps) {
  const [setting, setSetting] = useState<PageBgSetting | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [mode, setMode] = useState<string>('default')
  const [opacity, setOpacity] = useState<number>(0.86)
  const [pageUrl, setPageUrl] = useState('')
  const [uploading, setUploading] = useState(false)

  // 加载当前页的背景设置
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    getPageBackground(coursewareId, pageNum)
      .then(s => {
        if (cancelled) return
        setSetting(s)
        setMode(s.page_bg_mode || 'default')
        setOpacity(s.page_bg_opacity != null ? s.page_bg_opacity : 0.86)
        setPageUrl(s.page_bg_url || '')
      })
      .catch(() => { if (!cancelled) setMessage('❌ 加载本页背景设置失败') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [coursewareId, pageNum])

  // 保存页级背景设置
  const handleSave = async () => {
    if (saving || disabled) return
    setSaving(true)
    try {
      await setPageBackground(coursewareId, pageNum, {
        url: pageUrl,
        mode,
        opacity: mode === 'custom' ? opacity : (mode === 'none' ? 0 : null),
      })
      setMessage('✅ 第' + pageNum + '页背景设置已保存并秒换')
      onSwapped()
    } catch (e) {
      setMessage('❌ 保存失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setSaving(false) }
  }

  // 清除页级设置（回退跟随课件级）
  const handleClear = async () => {
    if (saving || disabled) return
    setSaving(true)
    try {
      await clearPageBackground(coursewareId, pageNum)
      setMode('default')
      setOpacity(0.86)
      setPageUrl('')
      setSetting({ page_bg_url: '', page_bg_opacity: null, page_bg_mode: 'default' })
      setMessage('✅ 第' + pageNum + '页背景已回退到跟随课件级')
      onSwapped()
    } catch (e) {
      setMessage('❌ 清除失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setSaving(false) }
  }

  // 上传单页背景图（复用课件资产上传端点，取返回的url补全域名作为背景图URL）
  const handleUploadPageBg = () => {
    const inp = document.createElement('input')
    inp.type = 'file'
    inp.accept = 'image/jpeg,image/png,image/webp'
    inp.onchange = async (ev) => {
      const f = (ev.target as HTMLInputElement).files?.[0]
      if (!f) return
      if (f.size > 5 * 1024 * 1024) { setMessage('❌ 图片不能超过5MB'); return }
      setUploading(true)
      try {
        // uploadCWImage 返回 { asset_id, url, file_name, file_size, mime_type }
        // url 是本地路径（如 /uploads/courseware-assets/...），需补全域名
        const { uploadCWImage } = await import('@/api/coursewares')
        const res = await uploadCWImage(coursewareId, pageNum, f)
        if (res && res.url) {
          const bgUrl = res.url.startsWith('http') ? res.url : 'https://workflow.pkuailab.com' + res.url
          setPageUrl(bgUrl)
          setMessage('✅ 图片已上传，点「保存」应用到本页')
        }
      } catch (e) {
        setMessage('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误'))
      } finally { setUploading(false) }
    }
    inp.click()
  }

  const hasPageOverride = setting && (setting.page_bg_url || setting.page_bg_mode !== 'default')

  if (loading) return <div style={{ marginTop: 14, padding: 12, fontSize: 12, color: C.textMuted }}>⏳ 加载第{pageNum}页背景设置...</div>

  return (
    <div style={{ marginTop: 14, padding: 14, borderRadius: 10, border: '1px solid #A78BFA', background: 'rgba(167,139,250,0.04)' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
        <div style={{ fontSize: 13, fontWeight: 600, color: '#7C3AED' }}>
          🎛️ 第{pageNum}页背景 {hasPageOverride ? <span style={{ fontSize: 11, color: '#059669', fontWeight: 400 }}>（已单独设置）</span> : <span style={{ fontSize: 11, color: C.textMuted, fontWeight: 400 }}>（跟随课件级）</span>}
        </div>
        {hasPageOverride && (
          <button onClick={handleClear} disabled={saving || disabled}
            style={{ padding: '4px 12px', borderRadius: 6, border: '1px solid #EF4444', background: 'rgba(239,68,68,0.06)', color: '#EF4444', fontSize: 11, cursor: (saving || disabled) ? 'default' : 'pointer' }}>
            ↩️ 回退到课件级
          </button>
        )}
      </div>

      {/* 蒙版模式选择 */}
      <div style={{ marginBottom: 10 }}>
        <div style={{ fontSize: 12, color: C.textSecondary, marginBottom: 6 }}>蒙版模式：</div>
        <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
          {BG_MODES.map(m => (
            <button key={m.key} onClick={() => { if (!disabled) setMode(m.key) }} disabled={disabled}
              style={{
                padding: '6px 14px', borderRadius: 8, fontSize: 12, cursor: disabled ? 'default' : 'pointer',
                border: '1px solid ' + (mode === m.key ? '#7C3AED' : C.border),
                background: mode === m.key ? 'rgba(124,58,237,0.08)' : '#fff',
                color: mode === m.key ? '#7C3AED' : C.textSecondary,
                fontWeight: mode === m.key ? 600 : 400,
              }}>
              {m.label}
            </button>
          ))}
        </div>
      </div>

      {/* 自定义透明度滑块（仅 custom 模式显示） */}
      {mode === 'custom' && (
        <div style={{ marginBottom: 10, padding: '10px 12px', borderRadius: 8, background: 'rgba(124,58,237,0.04)', border: '1px solid rgba(124,58,237,0.15)' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
            <span style={{ fontSize: 12, color: '#7C3AED' }}>蒙版透明度</span>
            <span style={{ fontSize: 12, fontWeight: 600, color: '#7C3AED' }}>{Math.round(opacity * 100)}%</span>
          </div>
          <input type="range" min={0} max={100} value={Math.round(opacity * 100)}
            onChange={e => setOpacity(Number(e.target.value) / 100)}
            disabled={disabled}
            style={{ width: '100%', accentColor: '#7C3AED' }} />
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 10, color: C.textMuted, marginTop: 2 }}>
            <span>0% 纯背景图</span>
            <span>50% 半透明</span>
            <span>100% 完全遮盖</span>
          </div>
        </div>
      )}

      {/* 单页背景图上传 */}
      <div style={{ marginBottom: 10 }}>
        <div style={{ fontSize: 12, color: C.textSecondary, marginBottom: 6 }}>本页专属背景图（可选，留空则使用课件级背景图）：</div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <input value={pageUrl} onChange={e => setPageUrl(e.target.value)} placeholder="背景图URL（留空=跟随课件级背景图）"
            disabled={disabled} style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid ' + C.border, fontSize: 12, outline: 'none', flex: 1 }} />
          <button onClick={handleUploadPageBg} disabled={disabled || uploading}
            style={{ padding: '6px 14px', borderRadius: 6, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 12, cursor: (disabled || uploading) ? 'default' : 'pointer', whiteSpace: 'nowrap' }}>
            {uploading ? '⏳' : '📤 上传'}
          </button>
        </div>
        {pageUrl && (
          <div style={{ marginTop: 6 }}>
            <img src={pageUrl.includes('aliyuncs.com') && !pageUrl.includes('?') ? pageUrl + '?x-oss-process=image/resize,w_300' : pageUrl}
              alt="本页背景预览" style={{ maxWidth: 200, maxHeight: 80, borderRadius: 6, border: '1px solid ' + C.border, objectFit: 'cover' }} />
          </div>
        )}
      </div>

      {/* 保存按钮 */}
      <button onClick={handleSave} disabled={saving || disabled}
        style={{
          padding: '8px 24px', borderRadius: 8, border: 'none', fontSize: 13, fontWeight: 600,
          background: (saving || disabled) ? '#E5E7EB' : 'linear-gradient(135deg, #7C3AED, #6D28D9)',
          color: (saving || disabled) ? '#9CA3AF' : '#fff',
          cursor: (saving || disabled) ? 'default' : 'pointer',
        }}>
        {saving ? '⏳ 保存中...' : '💾 保存本页背景设置'}
      </button>
    </div>
  )
}
