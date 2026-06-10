/**
 * BackgroundPicker.tsx — 课件背景图库选择器（批次2新建）
 *
 * 嵌入位置：课件工坊 Step 3「确认导航栏」页（封面预览下方）。
 * 交互：
 *   - 画廊展示全部可用图集（系统图库在前、我的个人集在后），每张卡片并排显示
 *     头图/内页两张缩略图；当前生效的集高亮。
 *   - 点选某集 → PUT 写快照 + 后端秒换全部已生成页（零token）→ 回调父组件
 *     loadCourseware() 重拉页面，封面预览立即换上新背景。
 *   - 「不使用背景」→ 清除选择，已生成页回退到模板自带背景（或无背景）。
 * 选中判定：后端只存URL快照不存集ID，故以 cover_bg_url === 集.cover_public_url 判定。
 */
import { useState, useEffect } from 'react'
import { listBackgroundSets, getCWBackground, setCWBackground, clearCWBackground } from '@/api/coursewares'
import type { CWBackgroundSet } from '@/api/coursewares'
import { C } from './workshopConstants'

interface Props {
  coursewareId: string
  /** 选/清背景成功后回调（父组件重拉课件与页面，刷新封面预览） */
  onSwapped: () => void
}

export default function BackgroundPicker({ coursewareId, onSwapped }: Props) {
  const [sets, setSets] = useState<CWBackgroundSet[]>([])
  const [coverBg, setCoverBg] = useState('')      // 当前生效的头图URL（''=未选背景）
  const [loading, setLoading] = useState(true)
  const [applying, setApplying] = useState('')    // 正在应用的集ID（'__clear__'=清除中, ''=空闲）
  const [message, setMessage] = useState('')

  // 进入即拉取：图集列表 + 课件当前选择（防卸载后 setState 用 cancelled 标志）
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
    if (applying) return
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

  const noneSelected = coverBg === ''

  return (
    <div style={{ marginTop: 16, padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#FAFAFA' }}>
      <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 4 }}>🖼️ 课件背景图（可选）</div>
      <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 12 }}>
        一组背景 = 封面头图 + 内页底纹。点选后封面立即换上新背景（无需重新生成），后续批量生成的内页自动使用内页底纹。
      </div>

      {loading && <div style={{ padding: '14px 0', fontSize: 13, color: C.textMuted, textAlign: 'center' }}>⏳ 背景图库加载中...</div>}

      {!loading && (
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
          {/* 不使用背景 */}
          <div onClick={() => { if (!applying && !noneSelected) apply(null) }}
            style={{
              width: 200, borderRadius: 10, overflow: 'hidden', cursor: (applying || noneSelected) ? 'default' : 'pointer',
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

          {/* 图集卡片：系统在前、我的在后（后端已排序） */}
          {sets.map(s => {
            const selected = !!coverBg && coverBg === s.cover_public_url
            const busy = applying === s.id
            return (
              <div key={s.id} onClick={() => { if (!applying && !selected) apply(s) }}
                style={{
                  width: 200, borderRadius: 10, overflow: 'hidden', cursor: (applying || selected) ? 'default' : 'pointer',
                  border: '2px solid ' + (selected ? C.primary : C.border),
                  background: selected ? C.primaryBg : '#fff', transition: 'all 200ms',
                }}>
                {/* 头图/内页双缩略图并排 */}
                <div style={{ display: 'flex', gap: 2, background: '#F3F4F6' }}>
                  <div style={{ flex: 1, position: 'relative' }}>
                    <img src={s.cover_public_url} alt="头图" loading="lazy"
                      style={{ width: '100%', height: 64, objectFit: 'cover', display: 'block' }} />
                    <span style={{ position: 'absolute', left: 4, bottom: 4, padding: '1px 6px', borderRadius: 4, background: 'rgba(0,0,0,0.55)', color: '#fff', fontSize: 10 }}>头图</span>
                  </div>
                  <div style={{ flex: 1, position: 'relative' }}>
                    <img src={s.content_public_url} alt="内页" loading="lazy"
                      style={{ width: '100%', height: 64, objectFit: 'cover', display: 'block' }} />
                    <span style={{ position: 'absolute', left: 4, bottom: 4, padding: '1px 6px', borderRadius: 4, background: 'rgba(0,0,0,0.55)', color: '#fff', fontSize: 10 }}>内页</span>
                  </div>
                </div>
                <div style={{ padding: '8px 10px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span style={{ fontSize: 13, fontWeight: 600, color: selected ? C.primary : C.textPrimary, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{s.name}</span>
                    <span style={{ flexShrink: 0, padding: '1px 6px', borderRadius: 4, fontSize: 10, fontWeight: 600, color: s.scope === 'system' ? '#2563EB' : '#B45309', background: s.scope === 'system' ? '#EFF6FF' : 'rgba(245,158,11,0.1)' }}>
                      {s.scope === 'system' ? '系统' : '我的'}
                    </span>
                  </div>
                  <div style={{ fontSize: 11, color: C.textMuted, marginTop: 2, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={s.description}>
                    {busy ? '⏳ 应用中...' : selected ? '✅ 当前使用中' : (s.description || '点击应用')}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {message && (
        <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, fontSize: 13, background: message.startsWith('❌') ? '#FEE2E2' : message.startsWith('✅') ? '#D1FAE5' : '#EFF6FF', color: message.startsWith('❌') ? '#DC2626' : message.startsWith('✅') ? '#059669' : '#2563EB' }}>{message}</div>
      )}
    </div>
  )
}
