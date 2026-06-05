/**
 * 课件视频分镜面板 — VideoStoryboardPanel.tsx
 *
 * 从 CoursewareWorkshopPage 抽出的视频生成区，仿「图片配图建议」卡片式交互，
 * 把单页视频改成【AI 分镜 → 逐镜两步生成】：
 *   ① 点「AI 写视频分镜」→ 后端按本页方案产出 1~N 个分镜(每镜 scene/首帧图提示词/视频提示词/台词)。
 *   ② 顶部「镜1/镜2/…」Tab 切换；每镜图片提示词、视频提示词分落在【可编辑文本框】，老师可改。
 *   ③ 每镜两步：先「生成本镜首帧图(16:9)」→ 再「生成本镜视频」；生成视频时自动用本镜首帧作参考图
 *      (图生视频)+ 写首帧溯源。视频按钮在出首帧前禁用，强制图生视频——彻底解决「首帧没被带进视频」。
 *   ④ 第 2 镜起默认勾选「参考第1镜首帧」，用第1镜首帧做图生图，保持全页各镜画风/人物一致。
 *
 * 多镜 = 多个独立 ~5 秒片段，都经 onAssetCreated 回传父组件进入本页视频列表，老师再用「视频编辑器」拼接。
 * 视频是异步任务：本组件自管轮询(queryVideoStatus)，完成后回传列表。
 * 另保留底部「手动直接生成(文生视频)」次要入口，给只想快出一段、不走分镜的场景。
 */
import { useState, useEffect, useRef } from 'react'
import { generateCWImage, generateCWVideo, getStoredVideoStoryboards, listPageAssets, queryVideoStatus, saveVideoStoryboards, suggestVideoPrompt } from '@/api/coursewares'
import type { VideoStoryboardItem, CoursewareAsset } from '@/api/coursewares'
import { C } from './courseware-workshop/workshopConstants'

// 单个分镜运行态(AI 物料 + 可编辑提示词 + 本镜首帧图信息)
interface ShotState {
  scene: string
  imagePrompt: string   // 可编辑：本镜首帧图提示词
  videoPrompt: string   // 可编辑：本镜图生视频提示词
  narration: string
  frameAssetId: string  // 本镜已生成首帧图资产ID（''=未生成）
  frameUrl: string      // 本镜已生成首帧图URL
  useShot1Ref: boolean  // 第2镜起：是否用第1镜首帧做参考图（默认true）
}

interface Props {
  coursewareId: string
  pageNum: number
  styleAnchorAssetId: string | null
  onAssetCreated: (asset: CoursewareAsset) => void  // 生成的首帧图/完成的视频回传，进入本页媒体列表
  onPreviewImage: (url: string) => void             // 点首帧缩略图 → 父组件大图预览
}

const PURPLE = '#7C3AED'

export default function VideoStoryboardPanel({ coursewareId, pageNum, styleAnchorAssetId, onAssetCreated, onPreviewImage }: Props) {
  const [shots, setShots] = useState<ShotState[]>([])
  const [activeIdx, setActiveIdx] = useState(0)
  const [suggesting, setSuggesting] = useState(false)
  const [frameGenIdx, setFrameGenIdx] = useState(-1)
  const [msg, setMsg] = useState('')

  // 视频生成 + 轮询（本组件自管；串行，一次只生成一镜视频）
  const [videoSubmitting, setVideoSubmitting] = useState(false)
  const [videoAssetId, setVideoAssetId] = useState('')
  const [videoPolling, setVideoPolling] = useState(false)
  const [videoResult, setVideoResult] = useState<{ url: string; duration: number; resolution: string } | null>(null)
  const submittedPromptRef = useRef('')
  const saveTimerRef = useRef<number | null>(null)

  const [manualPrompt, setManualPrompt] = useState('')

  // 切页/挂载：先清空分镜与生成态（防上一页串到新页，不自动调 AI）；
  // 再检测本页是否有「未完成(generating)的视频」——刷新/切走会中断轮询、留下卡在 generating 的孤儿任务，
  // 这里自动接管轮询(复用下方 queryVideoStatus 逻辑)直到 uploaded/failed，使视频不会再丢、刷新也能续上。
  useEffect(() => {
    setShots([]); setActiveIdx(0); setMsg('')
    setVideoResult(null); setVideoAssetId(''); setVideoPolling(false); setVideoSubmitting(false)
    setFrameGenIdx(-1); setManualPrompt('')
    if (!coursewareId || pageNum <= 0) return
    let cancelled = false
    listPageAssets(coursewareId, pageNum)
      .then(res => {
        if (cancelled) return
        const gen = (res.assets || []).filter(a => a.asset_type === 'video' && a.status === 'generating')
        if (gen.length > 0) {
          const latest = gen[gen.length - 1] // 列表按创建时间升序，取最近一条未完成视频
          submittedPromptRef.current = latest.generation_prompt || ''
          setVideoAssetId(latest.id)
          setVideoPolling(true)
          setMsg('⏳ 检测到本页有未完成的视频，正在继续等待生成结果（刷新/切页不会再丢）...')
        }
      })
      .catch(() => { /* 续轮询尽力而为，拉取失败就算了，不打扰用户 */ })
    return () => { cancelled = true }
  }, [pageNum, coursewareId])

  // 视频轮询：每5秒查一次，直到 uploaded / failed
  useEffect(() => {
    if (!videoPolling || !videoAssetId) return
    const timer = setInterval(async () => {
      try {
        const res = await queryVideoStatus(coursewareId, videoAssetId)
        if (res.status === 'uploaded') {
          setVideoPolling(false)
          setVideoResult({ url: res.video_url, duration: res.duration, resolution: res.resolution })
          setMsg('✅ ' + res.message)
          onAssetCreated({
            id: videoAssetId, courseware_id: coursewareId, page_id: null, placeholder_id: '',
            asset_type: 'video', generation_prompt: submittedPromptRef.current, oss_url: res.video_url,
            file_size: 0, mime_type: 'video/mp4', status: 'uploaded', created_at: new Date().toISOString(),
          })
        } else if (res.status === 'failed') {
          setVideoPolling(false); setMsg('❌ ' + res.message)
        } else {
          setMsg('⏳ 视频生成中...')
        }
      } catch (e) {
        setMsg('⚠️ 查询状态失败: ' + (e instanceof Error ? e.message : '未知错误'))
      }
    }, 5000)
    return () => clearInterval(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [videoPolling, videoAssetId, coursewareId])

  // 进页/挂载: 读取本页已存的视频分镜回填(后端在 AI 拆镜时已落库), 使刷新/切走再回来分镜不丢;
  //   库里没有则保持空, 等用户点「AI 写视频分镜」生成。
  useEffect(() => {
    if (!coursewareId || pageNum <= 0) return
    let cancelled = false
    getStoredVideoStoryboards(coursewareId, pageNum)
      .then(res => {
        if (cancelled) return
        const list = res.storyboards || []
        if (list.length > 0) {
          setShots(list.map((it, i) => ({
            scene: it.scene || ('镜' + (i + 1)),
            imagePrompt: it.image_prompt || '',
            videoPrompt: it.video_prompt || '',
            narration: it.narration || '',
            // 回填已存首帧关联: 恢复缩略图显示与视频按钮就绪态(库里无则为空串, 行为同未生成)
            frameAssetId: it.frame_asset_id || '', frameUrl: it.frame_url || '',
            useShot1Ref: i >= 1,
          })))
          setActiveIdx(0)
        }
      })
      .catch(() => { /* 读已存尽力而为, 失败保持空 */ })
    return () => { cancelled = true }
  }, [pageNum, coursewareId])

  const genBusy = suggesting || frameGenIdx >= 0 || videoSubmitting || videoPolling

  const handleSuggest = async () => {
    if (!coursewareId || pageNum <= 0 || genBusy) return
    if (shots.length > 0 && !confirm('重新拆分镜会覆盖当前各镜的提示词与首帧关联（已生成的图/视频仍在列表里）。确定？')) return
    setSuggesting(true); setMsg('🤖 AI 正在按本页方案拆分镜、写首帧图/视频提示词/台词...')
    try {
      const res = await suggestVideoPrompt(coursewareId, pageNum)
      const list: VideoStoryboardItem[] = res.storyboards || []
      const next: ShotState[] = list.map((it, i) => ({
        scene: it.scene || ('镜' + (i + 1)),
        imagePrompt: it.image_prompt || '',
        videoPrompt: it.video_prompt || '',
        narration: it.narration || '',
        frameAssetId: '', frameUrl: '',
        useShot1Ref: i >= 1,
      }))
      setShots(next); setActiveIdx(0)
      setMsg(next.length > 0
        ? '✅ AI 已拆成 ' + next.length + ' 个分镜，逐镜「先出首帧图 → 再生成视频」。提示词可在文本框直接改。'
        : '⚠️ AI 未返回分镜，可改用下方「手动直接生成」。')
    } catch (e) {
      setMsg('❌ 生成分镜失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setSuggesting(false) }
  }

  // 防抖保存当前分镜到库(仅手动改文本框/勾选时经 patchShot 触发; 切页/重置/AI拆镜/回填均不走此, 不会误存空覆盖别页)
  const scheduleSaveStoryboards = (next: ShotState[]) => {
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current)
    const cwId = coursewareId, pn = pageNum
    saveTimerRef.current = window.setTimeout(() => {
      saveVideoStoryboards(cwId, pn, next.map(s => ({
        scene: s.scene, image_prompt: s.imagePrompt, video_prompt: s.videoPrompt, narration: s.narration,
        // 首帧关联一并存库: 刷新/重进后回填缩略图与"可生视频"就绪态(空串=本镜尚无首帧)
        frame_asset_id: s.frameAssetId, frame_url: s.frameUrl,
      }))).catch(() => { /* 自动保存失败不打扰用户, 下次改动会再存 */ })
    }, 800)
  }

  const patchShot = (idx: number, patch: Partial<ShotState>) => {
    setShots(prev => {
      const next = prev.map((s, i) => (i === idx ? { ...s, ...patch } : s))
      scheduleSaveStoryboards(next)
      return next
    })
  }

  // ② 生成本镜首帧图(16:9)。第2镜起若勾选且第1镜已有首帧 → 用第1镜首帧做参考图
  const handleGenFrame = async (idx: number) => {
    const shot = shots[idx]
    if (!shot || genBusy) return
    const prompt = shot.imagePrompt.trim()
    if (!prompt) { setMsg('⚠️ 请先填写第' + (idx + 1) + '镜的首帧图提示词'); return }
    let refUrl: string | undefined = undefined
    if (idx >= 1 && shot.useShot1Ref && shots[0] && shots[0].frameUrl) refUrl = shots[0].frameUrl
    setFrameGenIdx(idx)
    setMsg('🖼️ 正在生成第' + (idx + 1) + '镜首帧图(16:9)，约10-30秒' + (refUrl ? '，参考第1镜保持一致' : '') + '...')
    try {
      const res = await generateCWImage(coursewareId, pageNum, prompt, undefined, '2560x1440', refUrl)
      patchShot(idx, { frameAssetId: res.asset_id, frameUrl: res.url })
      onAssetCreated({
        id: res.asset_id, courseware_id: coursewareId, page_id: null, placeholder_id: '',
        asset_type: 'image', generation_prompt: prompt, oss_url: res.url,
        file_size: 0, mime_type: 'image/png', status: 'uploaded', created_at: new Date().toISOString(),
      })
      setMsg('✅ 第' + (idx + 1) + '镜首帧图已生成' + (refUrl ? '（已参考第1镜，画风/人物一致）' : '') + '，可点「生成本镜视频」。')
    } catch (e) {
      setMsg('❌ 首帧图生成失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setFrameGenIdx(-1) }
  }

  // ③ 生成本镜视频：自动用本镜首帧作参考图(图生视频)+ 首帧资产ID溯源
  const handleGenVideo = async (idx: number) => {
    const shot = shots[idx]
    if (!shot || genBusy) return
    const vp = shot.videoPrompt.trim()
    if (!vp) { setMsg('⚠️ 请先填写第' + (idx + 1) + '镜的视频提示词'); return }
    if (!shot.frameUrl) { setMsg('⚠️ 请先生成第' + (idx + 1) + '镜首帧图，再生成视频'); return }
    submittedPromptRef.current = vp
    setVideoSubmitting(true); setMsg(''); setVideoResult(null)
    try {
      const res = await generateCWVideo(coursewareId, pageNum, vp, shot.frameUrl, shot.frameAssetId || undefined)
      setVideoAssetId(res.asset_id); setVideoSubmitting(false); setVideoPolling(true)
      setMsg('✅ ' + res.message + '（第' + (idx + 1) + '镜生成中，约30-120秒）')
    } catch (e) {
      setMsg('❌ 提交失败: ' + (e instanceof Error ? e.message : '未知错误')); setVideoSubmitting(false)
    }
  }

  const handleManualGen = async () => {
    if (!coursewareId || pageNum <= 0 || genBusy) return
    const p = manualPrompt.trim()
    if (!p) { setMsg('⚠️ 请先填写视频描述'); return }
    submittedPromptRef.current = p
    setVideoSubmitting(true); setMsg(''); setVideoResult(null)
    try {
      const res = await generateCWVideo(coursewareId, pageNum, p, undefined, undefined)
      setVideoAssetId(res.asset_id); setVideoSubmitting(false); setVideoPolling(true)
      setMsg('✅ ' + res.message)
    } catch (e) {
      setMsg('❌ 提交失败: ' + (e instanceof Error ? e.message : '未知错误')); setVideoSubmitting(false)
    }
  }

  const active = shots[activeIdx]
  const shot1HasFrame = !!(shots[0] && shots[0].frameUrl)
  const taField = { width: '100%', padding: '8px 10px', borderRadius: 6, border: '1px solid ' + C.border, fontSize: 12, resize: 'vertical' as const, outline: 'none', boxSizing: 'border-box' as const }

  return (
    <div style={{ padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff', marginTop: 16 }}>
      <div style={{ fontSize: 13, fontWeight: 600, color: PURPLE, marginBottom: 8 }}>🎬 AI 视频生成（分镜两步法）</div>

      {styleAnchorAssetId && (
        <div style={{ marginBottom: 10, padding: '6px 10px', borderRadius: 6, background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.3)', fontSize: 11, color: '#B45309', lineHeight: 1.6 }}>
          ⭐ 已设风格锚点：各镜首帧图会自动套用锚点画风与人物；第2镜起再参考第1镜首帧，全页更统一。
        </div>
      )}

      <button onClick={handleSuggest} disabled={genBusy}
        style={{ padding: '9px 14px', borderRadius: 8, border: '1px dashed ' + PURPLE, background: 'rgba(124,58,237,0.04)', color: PURPLE, fontSize: 13, fontWeight: 600, cursor: genBusy ? 'default' : 'pointer', width: '100%' }}>
        {suggesting ? '⏳ AI 拆分镜中...' : (shots.length > 0 ? '🔄 重新拆分镜（覆盖当前）' : '✨ AI 写视频分镜（按本页方案：首帧图+视频+台词）')}
      </button>

      {msg && (
        <div style={{ marginTop: 10, padding: '8px 12px', borderRadius: 8, fontSize: 12, background: msg.startsWith('❌') ? '#FEE2E2' : msg.startsWith('⚠') ? '#FEF3C7' : msg.startsWith('✅') ? '#D1FAE5' : '#EFF6FF', color: msg.startsWith('❌') ? '#DC2626' : msg.startsWith('⚠') ? '#D97706' : msg.startsWith('✅') ? '#059669' : '#2563EB' }}>{msg}</div>
      )}

      {shots.length > 0 && (
        <div style={{ marginTop: 12 }}>
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 10 }}>
            {shots.map((s, idx) => {
              const on = idx === activeIdx
              return (
                <button key={idx} onClick={() => setActiveIdx(idx)}
                  style={{ padding: '5px 12px', borderRadius: 8, border: '1px solid ' + (on ? PURPLE : C.border), background: on ? 'rgba(124,58,237,0.08)' : '#fff', color: on ? PURPLE : C.textSecondary, fontSize: 12, fontWeight: on ? 600 : 400, cursor: 'pointer' }}>
                  镜{idx + 1}{s.frameUrl ? ' ✅' : ''}
                </button>
              )
            })}
          </div>

          {active && (
            <div style={{ border: '1px solid ' + C.border, borderRadius: 10, padding: 12, background: '#FAFAFA' }}>
              <div style={{ fontSize: 12, fontWeight: 600, color: C.textPrimary, marginBottom: 10 }}>镜 {activeIdx + 1}{active.scene ? '：' + active.scene : ''}</div>

              <div style={{ fontSize: 11, color: PURPLE, fontWeight: 600, marginBottom: 4 }}>第一步 · 首帧图提示词（可改）</div>
              <textarea value={active.imagePrompt} onChange={e => patchShot(activeIdx, { imagePrompt: e.target.value })} rows={3} disabled={genBusy} style={taField} />
              {activeIdx >= 1 && (
                <label style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 6, fontSize: 11, color: shot1HasFrame ? C.textSecondary : '#9CA3AF', cursor: shot1HasFrame ? 'pointer' : 'default' }}>
                  <input type="checkbox" checked={active.useShot1Ref} disabled={!shot1HasFrame || genBusy}
                    onChange={e => patchShot(activeIdx, { useShot1Ref: e.target.checked })} style={{ width: 14, height: 14, accentColor: PURPLE }} />
                  参考第1镜首帧保持一致{shot1HasFrame ? '' : '（请先生成第1镜首帧）'}
                </label>
              )}
              <button onClick={() => handleGenFrame(activeIdx)} disabled={genBusy || !active.imagePrompt.trim()}
                style={{ marginTop: 8, padding: '8px 14px', borderRadius: 8, border: 'none', background: (!genBusy && active.imagePrompt.trim()) ? 'linear-gradient(135deg,#7C3AED,#6D28D9)' : '#E5E7EB', color: (!genBusy && active.imagePrompt.trim()) ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (!genBusy && active.imagePrompt.trim()) ? 'pointer' : 'default', width: '100%' }}>
                {frameGenIdx === activeIdx ? '⏳ 首帧图生成中...' : (active.frameUrl ? '🔄 重新生成本镜首帧图(16:9)' : '🖼️ 生成本镜首帧图(16:9)')}
              </button>
              {active.frameUrl && (
                <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 10 }}>
                  <img src={active.frameUrl} alt={'镜' + (activeIdx + 1) + '首帧'} onClick={() => onPreviewImage(active.frameUrl)} title="点击看大图"
                    style={{ width: 80, height: 45, objectFit: 'cover', borderRadius: 6, border: '2px solid ' + PURPLE, cursor: 'pointer', flexShrink: 0 }} />
                  <span style={{ fontSize: 11, color: '#059669' }}>✅ 本镜首帧已就绪，生成视频时以它为首帧</span>
                </div>
              )}

              <div style={{ fontSize: 11, color: PURPLE, fontWeight: 600, margin: '14px 0 4px' }}>第二步 · 视频提示词（可改，描述运镜/动作）</div>
              <textarea value={active.videoPrompt} onChange={e => patchShot(activeIdx, { videoPrompt: e.target.value })} rows={3} disabled={genBusy} style={taField} />
              {active.narration && (
                <div style={{ marginTop: 8, padding: 8, borderRadius: 6, border: '1px solid ' + C.border, background: '#fff' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                    <span style={{ fontSize: 11, fontWeight: 600, color: PURPLE }}>🗣️ 本镜台词</span>
                    <button onClick={() => { navigator.clipboard.writeText(active.narration).then(() => setMsg('📋 台词已复制')).catch(() => {}) }}
                      style={{ padding: '2px 8px', borderRadius: 5, border: '1px solid #059669', background: 'rgba(5,150,105,0.06)', color: '#059669', fontSize: 11, cursor: 'pointer' }}>📋 复制</button>
                  </div>
                  <div style={{ fontSize: 12, color: C.textPrimary, lineHeight: 1.6, whiteSpace: 'pre-wrap' }}>{active.narration}</div>
                </div>
              )}
              <button onClick={() => handleGenVideo(activeIdx)} disabled={genBusy || !active.videoPrompt.trim() || !active.frameUrl} title={!active.frameUrl ? '请先生成本镜首帧图' : ''}
                style={{ marginTop: 8, padding: '9px 14px', borderRadius: 8, border: 'none', background: (!genBusy && active.videoPrompt.trim() && active.frameUrl) ? 'linear-gradient(135deg,#7C3AED,#6D28D9)' : '#E5E7EB', color: (!genBusy && active.videoPrompt.trim() && active.frameUrl) ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (!genBusy && active.videoPrompt.trim() && active.frameUrl) ? 'pointer' : 'default', width: '100%' }}>
                {videoSubmitting ? '⏳ 提交中...' : videoPolling ? '⏳ 视频生成中（约30-120秒）...' : (active.frameUrl ? '🎬 生成本镜视频（用本镜首帧）' : '🎬 生成本镜视频（请先出首帧）')}
              </button>
            </div>
          )}
        </div>
      )}

      {videoResult && (
        <div style={{ marginTop: 12, borderRadius: 10, border: '1px solid ' + PURPLE, overflow: 'hidden' }}>
          <video src={videoResult.url} controls style={{ width: '100%', maxHeight: 360, display: 'block', background: '#000' }} />
          <div style={{ padding: '8px 12px', background: 'rgba(124,58,237,0.04)', fontSize: 12, color: '#6B7280' }}>
            🎬 时长 {videoResult.duration}秒 | 分辨率 {videoResult.resolution}（已进入下方视频列表，可上云盘/进编辑器拼接）
          </div>
        </div>
      )}

      <details style={{ marginTop: 14 }}>
        <summary style={{ fontSize: 12, color: C.textSecondary, cursor: 'pointer' }}>或：手动直接生成一段（文生视频，不分镜、无首帧）</summary>
        <div style={{ marginTop: 8 }}>
          <textarea value={manualPrompt} onChange={e => setManualPrompt(e.target.value)} rows={3} disabled={genBusy} style={taField}
            placeholder="直接描述想要的视频，例如：一位教师在讲台前微笑讲课，绿色黑板，教室明亮温馨" />
          <button onClick={handleManualGen} disabled={genBusy || !manualPrompt.trim()}
            style={{ marginTop: 6, padding: '8px 14px', borderRadius: 8, border: 'none', background: (!genBusy && manualPrompt.trim()) ? '#6B7280' : '#E5E7EB', color: (!genBusy && manualPrompt.trim()) ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (!genBusy && manualPrompt.trim()) ? 'pointer' : 'default', width: '100%' }}>
            {videoSubmitting ? '⏳ 提交中...' : videoPolling ? '⏳ 生成中...' : '🎬 直接生成视频'}
          </button>
        </div>
      </details>

      <div style={{ marginTop: 8, fontSize: 11, color: '#9CA3AF' }}>💡 每镜约5秒、720p。多镜分别生成后都在下方视频列表，用「视频编辑器」拼接成完整一段。</div>
    </div>
  )
}
