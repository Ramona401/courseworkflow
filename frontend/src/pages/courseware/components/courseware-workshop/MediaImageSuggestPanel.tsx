/**
 * MediaImageSuggestPanel.tsx — AI配图建议面板（批次5b从 MediaManagerPanel 拆出）
 *
 * 拆出范围：图片Tab左栏「✨ AI配图建议」整列 + 其专属的7个state/1个缓存ref/
 * 2个effect/6个处理函数。建议列表/勾选/Tab切换/批量生成进度全部自含，
 * 与父级仅通过6个回调prop交互，行为与拆分前完全一致。
 *
 * 与父级的接缝：
 *   - active：仅图片Tab激活且选中页有效时为true，控制自动拉建议的时机；
 *   - defaultSize：父级当前默认图片尺寸（新载入建议条目的初始尺寸）；
 *   - busyExternal：父级单张生成进行中，与本面板批量生成互斥（避免并发打爆豆包API）；
 *   - onFillPrompt：「→ 填入右侧」把提示词与尺寸填入父级生成框；
 *   - onAssetCreated：批量生成的新资产回传父级 mediaAssets 列表；
 *   - notify：全部提示消息路由回父级消息条（面板自身不渲染消息条）。
 *
 * 建议加载顺序（与拆分前一致）：本会话缓存 → 库已存(不调AI省token) → 调AI(自动写库)。
 * 切页清空建议防串页；切Tab来回时缓存即时恢复（行为等价于拆分前的清空+缓存回填）。
 */
import { useState, useEffect, useRef } from 'react'
import { generateCWImage, suggestImagePrompt, getStoredImageSuggestions } from '@/api/coursewares'
import type { ImagePromptSuggestion, CoursewareAsset } from '@/api/coursewares'
import { C } from './workshopConstants'
import { makeAsset } from './makeAsset'

interface Props {
  coursewareId: string
  pageNum: number
  /** 仅图片Tab激活且选中页有效时为 true */
  active: boolean
  /** 父级当前默认图片尺寸 */
  defaultSize: string
  /** 父级单张生成进行中（互斥禁用批量相关按钮） */
  busyExternal: boolean
  /** 把某条建议的提示词与尺寸填入父级生成框 */
  onFillPrompt: (prompt: string, size: string) => void
  /** 批量生成产出的新资产回传父级 */
  onAssetCreated: (asset: CoursewareAsset) => void
  /** 提示消息路由回父级消息条 */
  notify: (msg: string) => void
}

export default function MediaImageSuggestPanel({ coursewareId, pageNum, active, defaultSize, busyExternal, onFillPrompt, onAssetCreated, notify }: Props) {
  // ==================== 建议面板专属状态（5b自父面板整体迁入） ====================
  const [imgSuggestions, setImgSuggestions] = useState<(ImagePromptSuggestion & { size: string })[]>([])
  const [imgSuggestSelected, setImgSuggestSelected] = useState<Set<number>>(new Set())
  const [batchGenRunning, setBatchGenRunning] = useState(false)
  const [batchGenProgress, setBatchGenProgress] = useState({ done: 0, total: 0 })
  const [activeImgSuggestTab, setActiveImgSuggestTab] = useState(0)
  const [suggesting, setSuggesting] = useState(false)
  const [imgSuggestLoading, setImgSuggestLoading] = useState(false)
  // 本会话建议缓存：按页号缓存已载入的建议（切Tab/切页回来即时恢复，不重复消耗AI）
  const imgSuggestCache = useRef<Map<number, (ImagePromptSuggestion & { size: string })[]>>(new Map())

  // 切页清空建议列表/勾选/Tab（防串页残留）；切Tab无需清空——缓存命中会即时恢复
  useEffect(() => {
    setImgSuggestions([]); setImgSuggestSelected(new Set()); setActiveImgSuggestTab(0)
  }, [pageNum])

  // 进入图片Tab/切到未拉过的页时, 自动出本页配图建议
  // 顺序: 本会话缓存 → 库已存(不调AI省token) → 都没有才调AI(那条会自动写库)
  useEffect(() => {
    if (!active || !coursewareId || pageNum <= 0) return
    const cached = imgSuggestCache.current.get(pageNum)
    if (cached) { setImgSuggestions(cached); return }
    let cancelled = false
    setImgSuggestLoading(true)
    const pn = pageNum
    getStoredImageSuggestions(coursewareId, pn)
      .then(stored => {
        if (cancelled) return null
        const list = (stored.prompts || []).map(it => ({ ...it, size: defaultSize }))
        if (list.length > 0) {
          imgSuggestCache.current.set(pn, list)
          setImgSuggestions(list)
          notify('✅ 已载入本页已存的 ' + list.length + ' 条配图建议（未消耗AI）')
          return null
        }
        return suggestImagePrompt(coursewareId, pn)
      })
      .then(res => {
        if (cancelled || !res) return
        const list = (res.prompts || []).map(it => ({ ...it, size: defaultSize }))
        imgSuggestCache.current.set(pn, list)
        setImgSuggestions(list)
        notify(list.length > 0
          ? '✅ AI 建议本页配 ' + list.length + ' 张图，请在下方勾选或逐条填入生成框'
          : '⚠️ AI 未给出配图建议，可手动填写提示词生成')
      })
      .catch(e => { if (!cancelled) notify('❌ 生成配图建议失败: ' + (e instanceof Error ? e.message : '未知错误') + '（可点「重新生成配图建议」重试，或手动填写）') })
      .finally(() => { if (!cancelled) setImgSuggestLoading(false) })
    return () => { cancelled = true }
  }, [active, pageNum, coursewareId])

  // 「🔄重新生成」——重新调 AI 覆盖本页建议缓存并出卡片
  const handleSuggestImagePrompt = async () => {
    if (!coursewareId || pageNum <= 0 || suggesting) return
    setSuggesting(true); notify('🤖 AI 正在按本页方案重新撰写配图建议...')
    try {
      const res = await suggestImagePrompt(coursewareId, pageNum)
      const list = (res.prompts || []).map(it => ({ ...it, size: defaultSize }))
      imgSuggestCache.current.set(pageNum, list)
      setImgSuggestions(list)
      setImgSuggestSelected(new Set())
      notify(list.length > 0
        ? '✅ AI 建议本页配 ' + list.length + ' 张图，请在下方勾选或逐条填入生成框'
        : '⚠️ AI 未给出配图建议，可手动填写提示词生成')
    } catch (e) { notify('❌ 生成配图建议失败: ' + (e instanceof Error ? e.message : '未知错误')) }
    finally { setSuggesting(false) }
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

  // 把某条建议填入父级生成框(老师可微调后单独生成)
  const fillImgSuggest = (idx: number) => {
    const it = imgSuggestions[idx]
    if (!it) return
    onFillPrompt(it.prompt, it.size)
    notify('✅ 已把第 ' + (idx + 1) + ' 条提示词填入生成框，可微调后点「生成图片」')
  }

  // 批量生成勾选的多张图(串行, 避免并发打爆豆包 API; 逐张经 onAssetCreated 回传父级)
  const handleBatchGenImages = async () => {
    if (!coursewareId || pageNum <= 0 || batchGenRunning) return
    const idxList = Array.from(imgSuggestSelected).sort((a, b) => a - b)
    if (idxList.length === 0) { notify('⚠️ 请先勾选要生成的图片'); return }
    setBatchGenRunning(true)
    setBatchGenProgress({ done: 0, total: idxList.length })
    notify('🤖 开始批量生成 ' + idxList.length + ' 张图，逐张生成中...')
    let okCount = 0; let failCount = 0
    for (let k = 0; k < idxList.length; k++) {
      const it = imgSuggestions[idxList[k]]
      if (!it || !it.prompt.trim()) { failCount++; continue }
      setBatchGenProgress({ done: k, total: idxList.length })
      notify('🤖 正在生成第 ' + (k + 1) + '/' + idxList.length + ' 张：' + (it.caption || it.prompt.slice(0, 20)))
      try {
        const res = await generateCWImage(coursewareId, pageNum, it.prompt.trim(), undefined, it.size)
        onAssetCreated(makeAsset(coursewareId, { id: res.asset_id, oss_url: res.url, generation_prompt: it.prompt }))
        okCount++
      } catch { failCount++ }
    }
    setBatchGenProgress({ done: idxList.length, total: idxList.length })
    notify((failCount > 0 ? '⚠️' : '✅') + ' 批量生成完成：成功 ' + okCount + ' 张' + (failCount > 0 ? '，失败 ' + failCount + ' 张' : ''))
    setBatchGenRunning(false)
  }

  // ==================== JSX（与拆分前左栏逐行一致, 仅 setMediaMessage→notify / mediaGenerating→busyExternal） ====================
  return (
    <div style={{ flex: '1 1 320px', padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, flexWrap: 'wrap', gap: 6 }}>
        <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>✨ AI 配图建议</div>
        <button onClick={handleSuggestImagePrompt} disabled={suggesting || busyExternal || imgSuggestLoading}
          style={{ padding: '4px 12px', borderRadius: 6, border: '1px dashed #7C3AED', background: 'rgba(124,58,237,0.04)', color: '#7C3AED', fontSize: 12, fontWeight: 600, cursor: (suggesting || busyExternal || imgSuggestLoading) ? 'default' : 'pointer' }}>
          {suggesting ? '⏳ 撰写中...' : '🔄 重新生成'}
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
                const tabOn = idx === safeTab
                const checked = imgSuggestSelected.has(idx)
                return (
                  <div key={idx} onClick={() => setActiveImgSuggestTab(idx)}
                    style={{ display: 'flex', alignItems: 'center', gap: 5, padding: '5px 10px', borderRadius: 8, cursor: 'pointer', border: '1px solid ' + (tabOn ? '#7C3AED' : C.border), background: tabOn ? 'rgba(124,58,237,0.08)' : '#fff' }}>
                    <input type="checkbox" checked={checked} onChange={(e) => { e.stopPropagation(); toggleImgSuggest(idx) }} onClick={(e) => e.stopPropagation()} disabled={batchGenRunning}
                      style={{ width: 14, height: 14, cursor: batchGenRunning ? 'default' : 'pointer', accentColor: '#7C3AED' }} />
                    <span style={{ fontSize: 12, fontWeight: tabOn ? 600 : 400, color: tabOn ? '#7C3AED' : C.textSecondary }}>图{idx + 1}</span>
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
              <button onClick={handleBatchGenImages} disabled={batchGenRunning || busyExternal || imgSuggestSelected.size === 0}
                style={{ padding: '8px 16px', borderRadius: 8, border: 'none', background: (!batchGenRunning && !busyExternal && imgSuggestSelected.size > 0) ? 'linear-gradient(135deg, #7C3AED, #6D28D9)' : '#E5E7EB', color: (!batchGenRunning && !busyExternal && imgSuggestSelected.size > 0) ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (!batchGenRunning && !busyExternal && imgSuggestSelected.size > 0) ? 'pointer' : 'default' }}>
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
  )
}
