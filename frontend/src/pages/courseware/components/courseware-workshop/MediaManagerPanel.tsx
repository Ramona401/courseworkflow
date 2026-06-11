/**
 * MediaManagerPanel.tsx — 多媒体管理面板（批次W1从 CoursewareWorkshopPage 原样抽出）
 *
 * 抽出范围：Step5「🖼️ 多媒体管理」整块 + 其专属的20个state/3个effect/6个处理函数
 * + 大图预览弹窗 + 视频编辑器弹窗。逻辑与交互与抽出前完全一致（纯搬家）。
 *
 * 与父组件的接缝（仅风格锚点）：
 *   - 锚点设/清要更新父级 courseware 状态（页面顶部锚点条也依赖它），故 handleSetAnchor /
 *     handleClearAnchor 留在父级，通过 onSetAnchor/onClearAnchor 传入；
 *   - 这两个父级函数的成功/失败提示通过 notify 回调路由回本面板的 mediaMessage 消息条。
 *
 * 行为微变更（W1已知且接受）：本面板只在 Step5 渲染，离开 Step5 会卸载——
 * 面板内媒体状态（资产列表/建议缓存）随之重置，返回时 effect 自动重拉（建议优先读库存不耗AI）。
 */
import { useState, useEffect, useRef } from 'react'
import {
  generateCWImage, uploadCWImage, listPageAssets, deleteCWAsset,
  advancedConcatCWVideos, uploadCWVideo, uploadAssetToOSS,
  suggestImagePrompt, getStoredImageSuggestions,
} from '@/api/coursewares'
import type { ImagePromptSuggestion, CoursewareAsset, CoursewareDetail } from '@/api/coursewares'
import { C, CW_IMG_STYLES } from './workshopConstants'
import VideoStoryboardPanel from '../VideoStoryboardPanel'
import VideoEditorModal from '../VideoEditorModal'

interface Props {
  coursewareId: string
  /** 当前选中页（父级 buildPreviewNum 的只读别名，批次4b口径) */
  pageNum: number
  /** 父级课件对象（读 style_anchor_* 锚点字段） */
  courseware: CoursewareDetail
  /** 正在设为锚点的资产ID（''=空闲，父级状态） */
  anchorSetting: string
  /** 正在清除锚点（父级状态） */
  anchorClearing: boolean
  /** 设为锚点（父级实现：多模态提取VAOCI并落库；notify把提示路由回本面板消息条） */
  onSetAnchor: (assetId: string, notify: (msg: string) => void) => void
  /** 清除锚点（父级实现，带confirm；notify同上） */
  onClearAnchor: (notify: (msg: string) => void) => void
  /** W2: 图片/视频, 由Step5工作台顶级Tab控制(两Tab渲染同一组件实例, 切换不丢状态) */
  mediaTab: 'image' | 'video'
}

export default function MediaManagerPanel({ coursewareId, pageNum, courseware, anchorSetting, anchorClearing, onSetAnchor, onClearAnchor, mediaTab }: Props) {
  // ==================== 媒体管理状态（W1自主页面整体迁入） ====================
  const [mediaAssets, setMediaAssets] = useState<CoursewareAsset[]>([])
  const [mediaGenPrompt, setMediaGenPrompt] = useState('')
  const [mediaSize, setMediaSize] = useState('1920x1920')
  const [mediaGenerating, setMediaGenerating] = useState(false)
  const [mediaMessage, setMediaMessage] = useState('')
  const [mediaPreviewUrl, setMediaPreviewUrl] = useState('')
  const [mediaRefUrl, setMediaRefUrl] = useState('')  // 参考图URL（图生图）
  const [mediaPromptSuggesting, setMediaPromptSuggesting] = useState(false)
  // 图片多提示词: AI 返回的多条配图建议(每条含 caption/prompt/各自尺寸)
  const [imgSuggestions, setImgSuggestions] = useState<(ImagePromptSuggestion & { size: string })[]>([])
  const [imgSuggestSelected, setImgSuggestSelected] = useState<Set<number>>(new Set())
  const [batchGenRunning, setBatchGenRunning] = useState(false)
  const [batchGenProgress, setBatchGenProgress] = useState({ done: 0, total: 0 })
  const [activeImgSuggestTab, setActiveImgSuggestTab] = useState(0)
  const [mediaStyleKey, setMediaStyleKey] = useState('')
  // 遗留项②：显式记住上次追加的风格后缀确切文本，剥离时直接 replace 该串
  const [styleSuffixText, setStyleSuffixText] = useState('')
  const [imgSuggestLoading, setImgSuggestLoading] = useState(false)
  const imgSuggestCache = useRef<Map<number, (ImagePromptSuggestion & { size: string })[]>>(new Map())
  // W2: mediaTab 改为prop(见Props), 内部图片/视频切换按钮已移除
  const [editorOpen, setEditorOpen] = useState(false)
  const [editorExporting, setEditorExporting] = useState(false)

  // ==================== 三个媒体effect（W1自主页面整体迁入） ====================
  // 选中页变化时, 自动拉取当前页媒体资产并清空旧的（批次4b：切页即换页媒体, 防串页）
  useEffect(() => {
    if (!coursewareId || pageNum <= 0) { setMediaAssets([]); return }
    let cancelled = false
    setMediaAssets([])
    listPageAssets(coursewareId, pageNum)
      .then(res => { if (!cancelled) setMediaAssets(res.assets || []) })
      .catch(() => { if (!cancelled) setMediaAssets([]) })
    return () => { cancelled = true }
  }, [coursewareId, pageNum])

  // 切页或切 Tab 时清空 AI 多条建议列表 + 生成框 + 参考图(防串页残留)
  useEffect(() => {
    setImgSuggestions([]); setImgSuggestSelected(new Set())
    setMediaGenPrompt(''); setMediaRefUrl('')
    setActiveImgSuggestTab(0); setMediaStyleKey('')
  }, [pageNum, mediaTab])

  // 进入图片Tab/切到未拉过的页时, 自动出本页配图建议
  // 顺序: 本会话缓存 → 库已存(不调AI省token) → 都没有才调AI(那条会自动写库)
  useEffect(() => {
    if (mediaTab !== 'image' || !coursewareId || pageNum <= 0) return
    const cached = imgSuggestCache.current.get(pageNum)
    if (cached) { setImgSuggestions(cached); return }
    let cancelled = false
    setImgSuggestLoading(true)
    const pn = pageNum
    getStoredImageSuggestions(coursewareId, pn)
      .then(stored => {
        if (cancelled) return null
        const list = (stored.prompts || []).map(it => ({ ...it, size: mediaSize }))
        if (list.length > 0) {
          imgSuggestCache.current.set(pn, list)
          setImgSuggestions(list)
          setMediaMessage('✅ 已载入本页已存的 ' + list.length + ' 条配图建议（未消耗AI）')
          return null
        }
        return suggestImagePrompt(coursewareId, pn)
      })
      .then(res => {
        if (cancelled || !res) return
        const list = (res.prompts || []).map(it => ({ ...it, size: mediaSize }))
        imgSuggestCache.current.set(pn, list)
        setImgSuggestions(list)
        setMediaMessage(list.length > 0
          ? '✅ AI 建议本页配 ' + list.length + ' 张图，请在下方勾选或逐条填入生成框'
          : '⚠️ AI 未给出配图建议，可手动填写提示词生成')
      })
      .catch(e => { if (!cancelled) setMediaMessage('❌ 生成配图建议失败: ' + (e instanceof Error ? e.message : '未知错误') + '（可点「重新生成配图建议」重试，或手动填写）') })
      .finally(() => { if (!cancelled) setImgSuggestLoading(false) })
    return () => { cancelled = true }
  }, [mediaTab, pageNum, coursewareId])

  // ==================== 处理函数（W1自主页面整体迁入） ====================
  // 「🔄重新生成配图建议」——重新调 AI 覆盖本页建议缓存并出卡片
  const handleSuggestImagePrompt = async () => {
    if (!coursewareId || pageNum <= 0 || mediaPromptSuggesting) return
    setMediaPromptSuggesting(true); setMediaMessage('🤖 AI 正在按本页方案重新撰写配图建议...')
    try {
      const res = await suggestImagePrompt(coursewareId, pageNum)
      const list = (res.prompts || []).map(it => ({ ...it, size: mediaSize }))
      imgSuggestCache.current.set(pageNum, list)
      setImgSuggestions(list)
      setImgSuggestSelected(new Set())
      setMediaMessage(list.length > 0
        ? '✅ AI 建议本页配 ' + list.length + ' 张图，请在下方勾选或逐条填入生成框'
        : '⚠️ AI 未给出配图建议，可手动填写提示词生成')
    } catch (e) { setMediaMessage('❌ 生成配图建议失败: ' + (e instanceof Error ? e.message : '未知错误')) }
    finally { setMediaPromptSuggesting(false) }
  }

  // 切换某条建议的勾选状态
  const toggleImgSuggest = (idx: number) => {
    setImgSuggestSelected(prev => {
      const next = new Set(prev)
      if (next.has(idx)) next.delete(idx); else next.add(idx)
      return next
    })
  }

  // 修改某条建议的尺寸
  const setImgSuggestSize = (idx: number, size: string) => {
    setImgSuggestions(prev => prev.map((it, i) => i === idx ? { ...it, size } : it))
  }

  // 风格快选: 点风格把其描述作为后缀融入生成框(同风格再点=取消; 换风格=替换旧后缀)
  const toggleImgStyle = (key: string) => {
    const next = CW_IMG_STYLES.find(s => s.key === key)
    setMediaGenPrompt(prev => {
      let base = prev
      if (styleSuffixText) {
        if (base.includes('，' + styleSuffixText)) base = base.replace('，' + styleSuffixText, '')
        else if (base.includes(styleSuffixText)) base = base.replace(styleSuffixText, '')
        base = base.replace(/[，,\s]+$/, '')
      }
      if (key === mediaStyleKey) { setStyleSuffixText(''); return base }
      const desc = next ? next.desc : ''
      setStyleSuffixText(desc)
      return base ? base + '，' + desc : desc
    })
    setMediaStyleKey(key === mediaStyleKey ? '' : key)
  }

  // 把某条建议填入生成框(老师可微调后单独生成)
  const fillImgSuggest = (idx: number) => {
    const it = imgSuggestions[idx]
    if (!it) return
    setMediaGenPrompt(it.prompt)
    setMediaSize(it.size)
    setMediaMessage('✅ 已把第 ' + (idx + 1) + ' 条提示词填入生成框，可微调后点「生成图片」')
  }

  // 批量生成勾选的多张图(串行, 避免并发打爆豆包 API; 逐张塞入 mediaAssets)
  const handleBatchGenImages = async () => {
    if (!coursewareId || pageNum <= 0 || batchGenRunning) return
    const idxList = Array.from(imgSuggestSelected).sort((a, b) => a - b)
    if (idxList.length === 0) { setMediaMessage('⚠️ 请先勾选要生成的图片'); return }
    setBatchGenRunning(true)
    setBatchGenProgress({ done: 0, total: idxList.length })
    setMediaMessage('🤖 开始批量生成 ' + idxList.length + ' 张图，逐张生成中...')
    let okCount = 0; let failCount = 0
    for (let k = 0; k < idxList.length; k++) {
      const it = imgSuggestions[idxList[k]]
      if (!it || !it.prompt.trim()) { failCount++; continue }
      setBatchGenProgress({ done: k, total: idxList.length })
      setMediaMessage('🤖 正在生成第 ' + (k + 1) + '/' + idxList.length + ' 张：' + (it.caption || it.prompt.slice(0, 20)))
      try {
        const res = await generateCWImage(coursewareId, pageNum, it.prompt.trim(), undefined, it.size)
        setMediaAssets(prev => [{ id: res.asset_id, courseware_id: coursewareId, page_id: null, placeholder_id: '', asset_type: 'image', generation_prompt: it.prompt, oss_url: res.url, file_size: 0, mime_type: 'image/png', status: 'uploaded', created_at: new Date().toISOString() }, ...prev])
        okCount++
      } catch { failCount++ }
    }
    setBatchGenProgress({ done: idxList.length, total: idxList.length })
    setMediaMessage((failCount > 0 ? '⚠️' : '✅') + ' 批量生成完成：成功 ' + okCount + ' 张' + (failCount > 0 ? '，失败 ' + failCount + ' 张' : ''))
    setBatchGenRunning(false)
  }

  // ==================== JSX（与抽出前逐行一致, 仅 id!→coursewareId / mediaPageNum→pageNum / 锚点走props） ====================
  return (
    <div style={{ marginTop: 16, padding: 20, borderRadius: 12, border: '1px solid ' + C.border, background: '#FAFAFA' }}>
      <div style={{ fontSize: 15, fontWeight: 600, color: C.textPrimary, marginBottom: 12 }}>🖼️ 多媒体管理</div>



      {/* 媒体管理跟随上方大预览框选中页（批次4b口径） */}
      <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginBottom: 16, alignItems: 'center' }}>
        <span style={{ padding: '10px 14px', borderRadius: 8, background: C.primaryBg, color: C.primary, fontSize: 14, fontWeight: 600, whiteSpace: 'nowrap' }}>
          正在管理：第 {pageNum || '—'} 页的{mediaTab === 'video' ? '视频' : '图片'}
        </span>
        {pageNum > 0 && (
          <button
            onClick={async () => {
              if (!coursewareId || pageNum <= 0) return
              try {
                const res = await listPageAssets(coursewareId, pageNum)
                setMediaAssets(res.assets || [])
              } catch { setMediaAssets([]) }
            }}
            style={{ padding: '10px 16px', borderRadius: 8, border: '1px solid ' + C.border, background: '#fff', color: C.textSecondary, fontSize: 13, cursor: 'pointer' }}
          >
            🔄 刷新列表
          </button>
        )}
        <span style={{ fontSize: 12, color: C.textMuted }}>（切换上方预览页即可管理对应页的媒体）</span>
      </div>

      {pageNum > 0 && mediaTab === 'image' && (
        <>
        {/* 每页图列表顶部常驻锚点缩略图条——锚点是课件级，从 courseware 直接读，跨页无需请求 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12, padding: '8px 12px', borderRadius: 10, background: courseware.style_anchor_asset_id ? 'rgba(245,158,11,0.07)' : '#FAFAFA', border: '1px solid ' + (courseware.style_anchor_asset_id ? 'rgba(245,158,11,0.3)' : C.border) }}>
          {courseware.style_anchor_asset_id ? (
            <>
              {courseware.style_anchor_url && (
                <img src={courseware.style_anchor_url} alt="风格锚点" onClick={() => setMediaPreviewUrl(courseware.style_anchor_url)} style={{ width: 44, height: 44, objectFit: 'cover', borderRadius: 8, border: '2px solid #F59E0B', cursor: 'pointer', flexShrink: 0 }} title="点击查看锚点大图" />
              )}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 12, fontWeight: 600, color: '#B45309' }}>⭐ 当前风格锚点</div>
                <div style={{ fontSize: 11, color: C.textMuted, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>本页及全课件配图将自动参考此图，保持风格与人物一致</div>
              </div>
              <button onClick={() => onClearAnchor(setMediaMessage)} disabled={anchorClearing}
                style={{ padding: '4px 12px', borderRadius: 6, border: '1px solid ' + C.danger, background: 'transparent', color: C.danger, fontSize: 12, cursor: anchorClearing ? 'default' : 'pointer', whiteSpace: 'nowrap', flexShrink: 0 }}>
                {anchorClearing ? '清除中...' : '✕ 清除锚点'}
              </button>
            </>
          ) : (
            <div style={{ fontSize: 12, color: C.textMuted, lineHeight: 1.5 }}>
              💡 还未设风格锚点。在下方任意图片上点「⭐设为锚点」，之后全课件配图都会自动参考它，保持<b>画风与人物形象一致</b>。
            </div>
          )}
        </div>
        {/* 上半: 左右两栏 —— 左=AI配图建议(Tab切换), 右=生成图片(含风格快选) */}
        <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>

          {/* 左栏: AI 配图建议 */}
          <div style={{ flex: '1 1 320px', padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, flexWrap: 'wrap', gap: 6 }}>
              <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>✨ AI 配图建议</div>
              <button onClick={handleSuggestImagePrompt} disabled={mediaPromptSuggesting || mediaGenerating || imgSuggestLoading}
                style={{ padding: '4px 12px', borderRadius: 6, border: '1px dashed #7C3AED', background: 'rgba(124,58,237,0.04)', color: '#7C3AED', fontSize: 12, fontWeight: 600, cursor: (mediaPromptSuggesting || mediaGenerating || imgSuggestLoading) ? 'default' : 'pointer' }}>
                {mediaPromptSuggesting ? '⏳ 撰写中...' : '🔄 重新生成'}
              </button>
            </div>

            {imgSuggestLoading && (
              <div style={{ padding: '16px 12px', borderRadius: 8, background: 'rgba(124,58,237,0.04)', color: '#7C3AED', fontSize: 12, textAlign: 'center' }}>⏳ AI 正在按本页方案生成配图建议...</div>
            )}

            {!imgSuggestLoading && imgSuggestions.length === 0 && (
              <div style={{ padding: '16px 12px', borderRadius: 8, background: '#FAFAFA', color: C.textMuted, fontSize: 12, textAlign: 'center', lineHeight: 1.6 }}>暂无配图建议<br/>本页可能无图片占位（用SVG/CSS自绘）。<br/>可点「🔄 重新生成」，或在右侧手动描述生成。</div>
            )}

            {imgSuggestions.length >= 1 && (() => {
              const safeTab = activeImgSuggestTab < imgSuggestions.length ? activeImgSuggestTab : 0
              const it = imgSuggestions[safeTab]
              return (
              <div>
                <div style={{ fontSize: 12, color: C.textSecondary, marginBottom: 8 }}>AI 建议本页配 {imgSuggestions.length} 张图（{imgSuggestions.length > 1 ? '切 Tab 查看，勾选可批量生成' : '可填入右侧微调后生成'}）· 已勾选 {imgSuggestSelected.size}/{imgSuggestions.length}</div>

                {imgSuggestions.length > 1 && (
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 10 }}>
                    {imgSuggestions.map((s, idx) => {
                      const active = idx === safeTab
                      const checked = imgSuggestSelected.has(idx)
                      return (
                        <div key={idx} onClick={() => setActiveImgSuggestTab(idx)}
                          style={{ display: 'flex', alignItems: 'center', gap: 5, padding: '5px 10px', borderRadius: 8, cursor: 'pointer', border: '1px solid ' + (active ? '#7C3AED' : C.border), background: active ? 'rgba(124,58,237,0.08)' : '#fff' }}>
                          <input type="checkbox" checked={checked} onChange={(e) => { e.stopPropagation(); toggleImgSuggest(idx) }} onClick={(e) => e.stopPropagation()} disabled={batchGenRunning}
                            style={{ width: 14, height: 14, cursor: batchGenRunning ? 'default' : 'pointer', accentColor: '#7C3AED' }} />
                          <span style={{ fontSize: 12, fontWeight: active ? 600 : 400, color: active ? '#7C3AED' : C.textSecondary }}>图{idx + 1}</span>
                        </div>
                      )
                    })}
                  </div>
                )}

                {it && (
                  <div style={{ padding: 10, borderRadius: 8, border: '1px solid ' + C.border, background: '#FAFAFA' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                      {imgSuggestions.length === 1 && (
                        <input type="checkbox" checked={imgSuggestSelected.has(safeTab)} onChange={() => toggleImgSuggest(safeTab)} disabled={batchGenRunning}
                          style={{ width: 15, height: 15, cursor: batchGenRunning ? 'default' : 'pointer', accentColor: '#7C3AED' }} />
                      )}
                      <span style={{ fontSize: 12, fontWeight: 600, color: C.textPrimary }}>{'图 ' + (safeTab + 1)}{it.caption ? '：' + it.caption : ''}</span>
                    </div>
                    <div style={{ fontSize: 12, color: C.textSecondary, lineHeight: 1.6, whiteSpace: 'pre-wrap', maxHeight: 180, overflowY: 'auto' }}>{it.prompt}</div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 8, flexWrap: 'wrap' }}>
                      <span style={{ fontSize: 11, color: '#6B7280' }}>尺寸:</span>
                      <select value={it.size} onChange={e => setImgSuggestSize(safeTab, e.target.value)} disabled={batchGenRunning}
                        style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid ' + C.border, fontSize: 11 }}>
                        <option value="1920x1920">1:1 正方形</option>
                        <option value="2560x1440">16:9 宽屏</option>
                        <option value="3072x1280">2.4:1 超宽</option>
                        <option value="1440x2560">9:16 竖屏</option>
                      </select>
                      <button onClick={() => fillImgSuggest(safeTab)} disabled={batchGenRunning}
                        style={{ padding: '4px 10px', borderRadius: 6, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, cursor: batchGenRunning ? 'default' : 'pointer' }}>
                        → 填入右侧
                      </button>
                    </div>
                  </div>
                )}

                {batchGenRunning && batchGenProgress.total > 0 && (
                  <div style={{ marginTop: 10 }}>
                    <div style={{ height: 6, borderRadius: 3, background: '#EDE9FE', overflow: 'hidden' }}>
                      <div style={{ height: '100%', borderRadius: 3, transition: 'width 400ms', width: (batchGenProgress.done / batchGenProgress.total * 100) + '%', background: 'linear-gradient(90deg, #7C3AED, #6D28D9)' }} />
                    </div>
                    <div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>已完成 {batchGenProgress.done} / {batchGenProgress.total} 张</div>
                  </div>
                )}

                {imgSuggestions.length > 1 && (
                  <div style={{ display: 'flex', gap: 8, marginTop: 10, flexWrap: 'wrap' }}>
                    <button onClick={handleBatchGenImages} disabled={batchGenRunning || mediaGenerating || imgSuggestSelected.size === 0}
                      style={{ padding: '8px 16px', borderRadius: 8, border: 'none', background: (!batchGenRunning && !mediaGenerating && imgSuggestSelected.size > 0) ? 'linear-gradient(135deg, #7C3AED, #6D28D9)' : '#E5E7EB', color: (!batchGenRunning && !mediaGenerating && imgSuggestSelected.size > 0) ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (!batchGenRunning && !mediaGenerating && imgSuggestSelected.size > 0) ? 'pointer' : 'default' }}>
                      {batchGenRunning ? '⏳ 批量生成中...' : '🤖 批量生成勾选的 ' + imgSuggestSelected.size + ' 张'}
                    </button>
                    <button onClick={() => { setImgSuggestSelected(new Set()) }} disabled={batchGenRunning}
                      style={{ padding: '8px 14px', borderRadius: 8, border: '1px solid ' + C.border, background: '#fff', color: C.textSecondary, fontSize: 13, cursor: batchGenRunning ? 'default' : 'pointer' }}>
                      取消勾选
                    </button>
                  </div>
                )}
                <div style={{ marginTop: 6, fontSize: 11, color: '#9CA3AF' }}>💡 批量生成串行调用 AI，生成的图自动进入下方「本页图片」列表。</div>
              </div>
              )
            })()}
          </div>

          {/* 右栏: 生成图片(风格快选 + 生成框 + 尺寸 + 参考图 + 生成按钮) */}
          <div style={{ flex: '1 1 320px', padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
            <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>🤖 生成图片</div>

            {/* 风格快选: 已设锚点时禁用（优先级 锚点 > 快选预设） */}
            <div style={{ marginBottom: 8 }}>
              {courseware.style_anchor_asset_id ? (
                <div style={{ padding: '8px 12px', borderRadius: 8, background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.3)', fontSize: 11, color: '#B45309', lineHeight: 1.6 }}>
                  ⭐ 已设风格锚点，配图将<b>自动套用锚点风格</b>以保持全课件统一，无需再选画面风格（如需手动指定风格，请先在顶部清除锚点）。
                </div>
              ) : (
                <>
                  <div style={{ fontSize: 11, color: '#6B7280', marginBottom: 6 }}>🎨 画面风格（点选融入提示词，让出图更好看）:</div>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                    {CW_IMG_STYLES.map(s => {
                      const on = mediaStyleKey === s.key
                      return (
                        <button key={s.key} onClick={() => toggleImgStyle(s.key)} disabled={mediaGenerating}
                          style={{ padding: '5px 10px', borderRadius: 14, border: '1px solid ' + (on ? '#7C3AED' : C.border), background: on ? 'rgba(124,58,237,0.1)' : '#fff', color: on ? '#7C3AED' : C.textSecondary, fontSize: 11, fontWeight: on ? 600 : 400, cursor: mediaGenerating ? 'default' : 'pointer' }}>
                          {s.label}
                        </button>
                      )
                    })}
                  </div>
                </>
              )}
            </div>

            <textarea
              value={mediaGenPrompt}
              onChange={e => setMediaGenPrompt(e.target.value)}
              placeholder="从左侧建议点「→ 填入右侧」，或在此手动描述要生成的图片；可点上方风格按钮美化"
              rows={5}
              style={{ width: '100%', padding: '10px 12px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 13, resize: 'vertical', outline: 'none', boxSizing: 'border-box' }}
              disabled={mediaGenerating}
            />
            <div style={{ marginTop: 8 }}>
              <span style={{ fontSize: 12, color: '#6B7280', marginRight: 8 }}>图片比例:</span>
              <select value={mediaSize} onChange={e => setMediaSize(e.target.value)}
                style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid #E5E7EB', fontSize: 12 }}
                disabled={mediaGenerating}>
                <option value="1920x1920">1:1 正方形</option>
                <option value="2560x1440">16:9 宽屏</option>
                <option value="3072x1280">2.4:1 超宽</option>
                <option value="1440x2560">9:16 竖屏</option>
              </select>
            </div>
            <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <span style={{ fontSize: 12, color: '#6B7280' }}>参考图:</span>
              {mediaRefUrl ? (
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <img src={mediaRefUrl} alt="参考图" style={{ width: 40, height: 40, objectFit: 'cover', borderRadius: 6, border: '2px solid #7C3AED' }} />
                  <span style={{ fontSize: 11, color: '#7C3AED' }}>已选择参考图</span>
                  <button onClick={() => setMediaRefUrl('')} style={{ padding: '2px 8px', borderRadius: 4, border: '1px solid #EF4444', background: 'transparent', color: '#EF4444', fontSize: 11, cursor: 'pointer' }}>取消</button>
                </div>
              ) : (
                <>
                <span style={{ fontSize: 11, color: '#9CA3AF' }}>无</span>
                <button onClick={() => {
                  const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/*'
                  inp.onchange = async (ev) => {
                    const f = (ev.target as HTMLInputElement).files?.[0]
                    if (!f || !coursewareId || pageNum <= 0) return
                    if (f.size > 5 * 1024 * 1024) { setMediaMessage('❌ 参考图不能超过5MB'); return }
                    setMediaMessage('⏳ 上传参考图中...')
                    try {
                      const res = await uploadCWImage(coursewareId, pageNum, f)
                      setMediaRefUrl(res.url)
                      setMediaAssets(prev => [{ id: res.asset_id, courseware_id: coursewareId, page_id: null, placeholder_id: '', asset_type: 'image', generation_prompt: '', oss_url: res.url, file_size: res.file_size, mime_type: res.mime_type, status: 'uploaded', created_at: new Date().toISOString() }, ...prev])
                      setMediaMessage('✅ 参考图上传成功，已自动选为参考图')
                    } catch (e) { setMediaMessage('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                  }; inp.click()
                }} style={{ padding: '3px 10px', borderRadius: 5, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, cursor: 'pointer' }}>📤 上传参考图</button>
                </>
              )}
            </div>
            <button
              onClick={async () => {
                if (!coursewareId || pageNum <= 0 || !mediaGenPrompt.trim() || mediaGenerating) return
                setMediaGenerating(true); setMediaMessage('')
                try {
                  const res = await generateCWImage(coursewareId, pageNum, mediaGenPrompt.trim(), undefined, mediaSize, mediaRefUrl || undefined)
                  setMediaMessage('✅ 图片生成成功！')
                  setMediaAssets(prev => [{ id: res.asset_id, courseware_id: coursewareId, page_id: null, placeholder_id: '', asset_type: 'image', generation_prompt: mediaGenPrompt, oss_url: res.url, file_size: 0, mime_type: 'image/png', status: 'uploaded', created_at: new Date().toISOString() }, ...prev])
                } catch (e) { setMediaMessage('❌ 生成失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                finally { setMediaGenerating(false) }
              }}
              disabled={mediaGenerating || !mediaGenPrompt.trim()}
              style={{ marginTop: 8, padding: '10px 20px', borderRadius: 8, border: 'none', background: mediaGenPrompt.trim() && !mediaGenerating ? 'linear-gradient(135deg, #7C3AED, #6D28D9)' : '#E5E7EB', color: mediaGenPrompt.trim() && !mediaGenerating ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: mediaGenPrompt.trim() && !mediaGenerating ? 'pointer' : 'default', width: '100%' }}
            >
              {mediaGenerating ? '⏳ AI生成中（约10-30秒）...' : '🤖 生成图片'}
            </button>
          </div>
        </div>

        {/* 下半: 手动上传(通栏) */}
        <div style={{ marginTop: 16, padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>📤 手动上传图片</div>
          <div style={{ padding: '20px 16px', borderRadius: 8, border: '2px dashed ' + C.border, textAlign: 'center', cursor: 'pointer', background: '#FAFAFA' }}
            onClick={() => { const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/*'; inp.onchange = async (ev) => { const f = (ev.target as HTMLInputElement).files?.[0]; if (!f || !coursewareId) return; if (f.size > 5 * 1024 * 1024) { setMediaMessage('❌ 图片不能超过5MB'); return } setMediaGenerating(true); setMediaMessage(''); try { const res = await uploadCWImage(coursewareId, pageNum, f); setMediaMessage('✅ 上传成功！'); setMediaAssets(prev => [{ id: res.asset_id, courseware_id: coursewareId, page_id: null, placeholder_id: '', asset_type: 'image', generation_prompt: '', oss_url: res.url, file_size: res.file_size, mime_type: res.mime_type, status: 'uploaded', created_at: new Date().toISOString() }, ...prev]) } catch (e) { setMediaMessage('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setMediaGenerating(false) } }; inp.click() }}
          >
            <div style={{ fontSize: 28, marginBottom: 6 }}>📷</div>
            <div style={{ fontSize: 13, color: C.textSecondary }}>点击选择图片</div>
            <div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>支持 JPG/PNG/WEBP/GIF/SVG，最大5MB</div>
          </div>
        </div>
        </>
      )}

      {/* 提示消息 */}
      {mediaTab === 'image' && mediaMessage && <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, background: mediaMessage.startsWith('❌') ? '#FEE2E2' : '#D1FAE5', color: mediaMessage.startsWith('❌') ? '#DC2626' : '#059669', fontSize: 13 }}>{mediaMessage}</div>}

      {/* 已上传图片列表 */}
      {(mediaAssets.filter(a => a.asset_type === 'image').length > 0 || courseware.style_anchor_asset_id) && mediaTab === 'image' && (
        <div style={{ marginTop: 16 }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>📎 第 {pageNum} 页的图片（{mediaAssets.filter(a => a.asset_type === 'image').length}张）</div>
          <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            {/* 锚点图置顶特殊项（读 courseware 会话缓存，跨页常驻）；下方 .map 已排除锚点图本身避免重复 */}
            {courseware.style_anchor_asset_id && courseware.style_anchor_url && (
              <div key={'anchor-pinned'} style={{ width: 'calc(25% - 9px)', minWidth: 180, borderRadius: 10, border: '2px solid #F59E0B', overflow: 'hidden', background: 'rgba(245,158,11,0.04)' }}>
                <img src={courseware.style_anchor_url} alt="风格锚点" onClick={() => setMediaPreviewUrl(courseware.style_anchor_url)} style={{ width: '100%', height: 140, objectFit: 'cover', display: 'block', cursor: 'pointer' }} title="点击查看锚点大图" />
                <div style={{ padding: '8px 10px' }}>
                  <div style={{ fontSize: 11, color: '#B45309', fontWeight: 600, marginBottom: 6 }}>★ 课件风格锚点</div>
                  <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                    <button
                      onClick={() => { setMediaRefUrl(courseware.style_anchor_url); setMediaMessage('✅ 已选锚点图为参考图，本次生成将参考锚点风格与人物') }}
                      style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, cursor: 'pointer' }}
                    >参考</button>
                    <button onClick={() => onClearAnchor(setMediaMessage)} disabled={anchorClearing}
                      style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid ' + C.danger, background: 'transparent', color: C.danger, fontSize: 11, cursor: anchorClearing ? 'default' : 'pointer' }}
                    >{anchorClearing ? '清除中' : '✕清除锚点'}</button>
                  </div>
                </div>
              </div>
            )}
            {mediaAssets.filter(a => a.asset_type === 'image' && a.id !== courseware.style_anchor_asset_id).map(asset => (
              <div key={asset.id} style={{ width: 'calc(25% - 9px)', minWidth: 180, borderRadius: 10, border: '1px solid ' + C.border, overflow: 'hidden', background: '#fff' }}>
                <img src={asset.oss_url} alt="课件图片" onClick={() => setMediaPreviewUrl(asset.oss_url)} style={{ width: '100%', height: 140, objectFit: 'cover', display: 'block', cursor: 'pointer' }} title="点击查看大图" />
                <div style={{ padding: '8px 10px' }}>
                  <div style={{ fontSize: 11, color: C.textMuted, marginBottom: 6 }}>{asset.generation_prompt ? '🤖 AI生成' : '📤 手动上传'}{asset.public_oss_url ? ' · ☁️已上云' : ''}</div>
                  <div style={{ display: 'flex', gap: 4 }}>
                    {courseware.style_anchor_asset_id === asset.id ? (
                      <span style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #F59E0B', background: 'rgba(245,158,11,0.12)', color: '#B45309', fontSize: 11, fontWeight: 600, whiteSpace: 'nowrap' }} title="当前风格锚点">★ 锚点</span>
                    ) : (
                      <button
                        onClick={() => onSetAnchor(asset.id, setMediaMessage)}
                        disabled={!!anchorSetting}
                        title="设为风格锚点：后续配图将自动参考此图风格，保持全课件视觉统一"
                        style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #F59E0B', background: 'rgba(245,158,11,0.06)', color: '#B45309', fontSize: 11, cursor: anchorSetting ? 'default' : 'pointer', whiteSpace: 'nowrap' }}
                      >{anchorSetting === asset.id ? '⏳设置中' : '⭐设为锚点'}</button>
                    )}
                    <button
                      onClick={() => { setMediaRefUrl(asset.oss_url); setMediaMessage('✅ 已选为参考图，AI将参考此图风格生成新图片') }}
                      style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, cursor: 'pointer' }}
                    >参考</button>
                    {asset.public_oss_url ? (
                      <button
                        onClick={async () => {
                          try {
                            await navigator.clipboard.writeText(asset.public_oss_url!)
                            setMediaMessage('📋 云盘链接已复制到剪贴板')
                          } catch { setMediaMessage('❌ 复制失败，请手动复制') }
                        }}
                        style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #059669', background: 'rgba(5,150,105,0.06)', color: '#059669', fontSize: 11, cursor: 'pointer' }}
                        title="已上传云盘，点击复制公网链接"
                      >📋 复制链接</button>
                    ) : (
                      <button
                        onClick={async () => {
                          if (!coursewareId) return
                          setMediaMessage('⏳ 正在上传到云盘...')
                          try {
                            const res = await uploadAssetToOSS(coursewareId, asset.id)
                            await navigator.clipboard.writeText(res.oss_public_url)
                            setMediaAssets(prev => prev.map(a => a.id === asset.id ? { ...a, public_oss_url: res.oss_public_url } : a))
                            setMediaMessage('✅ 已上传云盘，链接已复制到剪贴板')
                          } catch (e) { setMediaMessage('❌ 上传云盘失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                        }}
                        style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #0891B2', background: 'rgba(8,145,178,0.06)', color: '#0891B2', fontSize: 11, cursor: 'pointer' }}
                        title="上传到云盘获取公网链接"
                      >☁️云盘</button>
                    )}
                    <button
                      onClick={async () => {
                        if (!coursewareId) return
                        const warnMsg = asset.public_oss_url
                          ? '⚠️ 这张图片已上传云盘。删除将同时移除云盘副本，若课件中已使用该云盘链接，图片将无法显示，且不可恢复。确定删除？'
                          : '确定删除这张图片？'
                        if (!confirm(warnMsg)) return
                        try {
                          await deleteCWAsset(coursewareId, asset.id)
                          setMediaAssets(prev => prev.filter(a => a.id !== asset.id))
                          setMediaMessage('✅ 已删除')
                        } catch (e) { setMediaMessage('❌ 删除失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                      }}
                      style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #EF4444', background: 'rgba(239,68,68,0.06)', color: '#EF4444', fontSize: 11, cursor: 'pointer' }}
                    >删除</button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 视频生成区: VideoStoryboardPanel(AI分镜两步法) */}
      {mediaTab === 'video' && pageNum > 0 && (
        <VideoStoryboardPanel
          coursewareId={coursewareId}
          pageNum={pageNum}
          styleAnchorAssetId={courseware.style_anchor_asset_id}
          onAssetCreated={(asset) => setMediaAssets(prev => prev.some(a => a.id === asset.id) ? prev : [asset, ...prev])}
          onPreviewImage={(url) => setMediaPreviewUrl(url)}
        />
      )}

      {/* 视频列表 + 编辑器入口（视频Tab下） */}
      {mediaTab === 'video' && pageNum > 0 && mediaAssets.filter(a => a.asset_type === 'video').length > 0 && (
        <div style={{ marginTop: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
            <span style={{ fontSize: 13, fontWeight: 600, color: '#7C3AED' }}>🎬 第 {pageNum} 页的视频（{mediaAssets.filter(a => a.asset_type === 'video').length}个）</span>
            <button onClick={() => setEditorOpen(true)} style={{ padding: '6px 14px', borderRadius: 8, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 12, fontWeight: 600, cursor: 'pointer' }}>🎬 打开视频编辑器</button>
          </div>
          <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            {mediaAssets.filter(a => a.asset_type === 'video').map(asset => (
              <div key={asset.id} style={{ width: 240, borderRadius: 10, border: '1px solid ' + C.border, overflow: 'hidden', background: '#fff' }}>
                <video src={asset.oss_url} controls style={{ width: '100%', height: 135, display: 'block', background: '#000', objectFit: 'contain' }} />
                <div style={{ padding: '8px 10px' }}>
                  <div style={{ fontSize: 11, color: C.textMuted, marginBottom: 4 }}>{asset.generation_prompt ? '🤖 ' + asset.generation_prompt.slice(0,25) + (asset.generation_prompt.length > 25 ? '...' : '') : '📤 手动上传'}{asset.public_oss_url ? ' · ☁️已上云' : ''}</div>
                  <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                    {asset.public_oss_url ? (
                      <button onClick={async () => {
                        try {
                          await navigator.clipboard.writeText(asset.public_oss_url!)
                          setMediaMessage('📋 云盘链接已复制到剪贴板')
                        } catch { setMediaMessage('❌ 复制失败，请手动复制') }
                      }} style={{ padding: '3px 8px', borderRadius: 5, border: '1px solid #059669', background: 'rgba(5,150,105,0.06)', color: '#059669', fontSize: 10, cursor: 'pointer' }} title="已上传云盘，点击复制公网链接">📋 复制链接</button>
                    ) : (
                      <button onClick={async () => {
                        if (!coursewareId) return
                        setMediaMessage('⏳ 正在上传到云盘...')
                        try {
                          const res = await uploadAssetToOSS(coursewareId, asset.id)
                          await navigator.clipboard.writeText(res.oss_public_url)
                          setMediaAssets(prev => prev.map(a => a.id === asset.id ? { ...a, public_oss_url: res.oss_public_url } : a))
                          setMediaMessage('✅ 已上传云盘，链接已复制到剪贴板')
                        } catch (e) { setMediaMessage('❌ 上传云盘失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                      }} style={{ padding: '3px 8px', borderRadius: 5, border: '1px solid #0891B2', background: 'rgba(8,145,178,0.06)', color: '#0891B2', fontSize: 10, cursor: 'pointer' }} title="上传到云盘获取公网链接">☁️云盘</button>
                    )}
                    <button onClick={async () => {
                        if (!coursewareId) return
                        const warnMsg = asset.public_oss_url
                          ? '⚠️ 这个视频已上传云盘。删除将同时移除云盘副本，若课件中已使用该云盘链接，视频将无法播放，且不可恢复。确定删除？'
                          : '确定删除此视频？'
                        if (!confirm(warnMsg)) return
                        try { await deleteCWAsset(coursewareId, asset.id); setMediaAssets(prev => prev.filter(a => a.id !== asset.id)); setMediaMessage('✅ 已删除') } catch (e) { setMediaMessage('❌ 删除失败: ' + (e instanceof Error ? e.message : '')) }
                      }} style={{ padding: '3px 8px', borderRadius: 5, border: '1px solid #EF4444', background: 'rgba(239,68,68,0.06)', color: '#EF4444', fontSize: 10, cursor: 'pointer' }}>删除</button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 视频Tab下也展示提示消息 */}
      {mediaTab === 'video' && mediaMessage && <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, background: mediaMessage.startsWith('❌') ? '#FEE2E2' : '#D1FAE5', color: mediaMessage.startsWith('❌') ? '#DC2626' : '#059669', fontSize: 13 }}>{mediaMessage}</div>}

      {/* 图片大图预览弹窗 */}
      {mediaPreviewUrl && (
        <div onClick={() => setMediaPreviewUrl('')} style={{ position: 'fixed', inset: 0, zIndex: 99990, background: 'rgba(0,0,0,0.75)', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'zoom-out' }}>
          <img src={mediaPreviewUrl} alt="大图预览" style={{ maxWidth: '90vw', maxHeight: '90vh', borderRadius: 12, boxShadow: '0 8px 40px rgba(0,0,0,0.5)' }} />
          <button onClick={(e) => { e.stopPropagation(); setMediaPreviewUrl('') }} style={{ position: 'absolute', top: 24, right: 24, width: 40, height: 40, borderRadius: '50%', border: 'none', background: 'rgba(255,255,255,0.2)', color: '#fff', fontSize: 20, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>✕</button>
        </div>
      )}

      {/* 视频编辑器弹窗（类剪映多片段时间轴编辑；fixed定位，渲染位置不影响视觉） */}
      {editorOpen && (
        <VideoEditorModal
          coursewareId={coursewareId}
          videos={mediaAssets.filter(a => a.asset_type === 'video' && a.oss_url).map(a => ({
            id: a.id,
            url: a.oss_url,
            label: a.generation_prompt || a.oss_url.split('/').pop()?.slice(0, 30) || '视频',
          }))}
          exporting={editorExporting}
          onClose={() => setEditorOpen(false)}
          onUploadVideo={async (file, onProgress) => {
            if (!coursewareId || pageNum <= 0) return null
            const res = await uploadCWVideo(coursewareId, pageNum, file, onProgress)
            const newAsset = {
              id: res.asset_id, courseware_id: coursewareId, page_id: null, placeholder_id: '',
              asset_type: 'video' as const, generation_prompt: file.name,
              oss_url: res.url, file_size: res.file_size, mime_type: res.mime_type,
              status: 'uploaded', created_at: new Date().toISOString(),
            }
            setMediaAssets(prev => [newAsset, ...prev])
            return { id: res.asset_id, url: res.url, label: file.name }
          }}
          onExport={async (clips) => {
            if (!coursewareId || editorExporting) return
            setEditorExporting(true); setMediaMessage('')
            try {
              const res = await advancedConcatCWVideos(coursewareId, clips)
              setMediaMessage('✅ ' + res.message)
              setMediaAssets(prev => [{
                id: res.asset_id, courseware_id: coursewareId, page_id: null, placeholder_id: '',
                asset_type: 'video', generation_prompt: '编辑导出 ' + clips.length + '个片段',
                oss_url: res.url, file_size: 0, mime_type: 'video/mp4', status: 'uploaded',
                created_at: new Date().toISOString(),
              }, ...prev])
              setEditorOpen(false)
            } catch (e) {
              setMediaMessage('❌ 导出失败: ' + (e instanceof Error ? e.message : ''))
            } finally { setEditorExporting(false) }
          }}
        />
      )}
    </div>
  )
}
