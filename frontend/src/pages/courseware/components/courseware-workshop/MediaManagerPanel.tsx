/**
 * MediaManagerPanel.tsx — 多媒体管理面板·编排壳（批次5b模块化拆分后）
 *
 * 历史：批次W1从 CoursewareWorkshopPage 整体抽出；批次5a加入红色删除确认弹窗
 * (DangerConfirmModal) 与 makeAsset 工厂；批次5b按600行红线拆分为四个模块：
 *   - makeAsset.ts               : 本地资产对象工厂（共享）
 *   - MediaImageSuggestPanel.tsx : AI配图建议整列（建议状态/缓存/批量生成自含）
 *   - MediaAssetCards.tsx        : ImageAssetList/VideoAssetList 资产卡片列表
 *                                  （云盘上传/复制链接逻辑去重为 CloudActions）
 *   - 本文件                      : 编排壳——共享状态(mediaAssets/消息条/参考图/
 *                                  弹窗/编辑器)+单张生成+手动上传+S-V2导出链
 *
 * 与父组件(CoursewareWorkshopPage)的接缝（仅风格锚点，与拆分前一致）：
 *   - 锚点设/清要更新父级 courseware 状态，故 handleSetAnchor/handleClearAnchor
 *     留在父级经 onSetAnchor/onClearAnchor 传入；提示经 notify 路由回本面板消息条。
 *
 * S-V2：onExport 三步串行链（advancedConcat → burnIn可选 → mixNarration可选），
 * 每步失败自动降级保留上一步可用成片，与三选项导出弹窗配套。
 */
import { useState, useEffect } from 'react'
import {
  generateCWImage, uploadCWImage, listPageAssets, deleteCWAsset,
  advancedConcatCWVideos, uploadCWVideo, burnInSubtitle,
} from '@/api/coursewares'
import type { CoursewareAsset, CoursewareDetail } from '@/api/coursewares'
import { mixNarrationCWVideo } from '@/api/coursewares.media'
import { C, CW_IMG_STYLES } from './workshopConstants'
import VideoStoryboardPanel from '../VideoStoryboardPanel'
import VideoEditorModal from '../VideoEditorModal'
import DangerConfirmModal from './DangerConfirmModal'
import MediaImageSuggestPanel from './MediaImageSuggestPanel'
import { ImageAssetList, VideoAssetList } from './MediaAssetCards'
import { makeAsset } from './makeAsset'

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
  // ==================== 共享状态（资产列表/消息条/单张生成/参考图/弹窗/编辑器） ====================
  const [mediaAssets, setMediaAssets] = useState<CoursewareAsset[]>([])
  const [mediaGenPrompt, setMediaGenPrompt] = useState('')
  const [mediaSize, setMediaSize] = useState('1920x1920')
  const [mediaGenerating, setMediaGenerating] = useState(false)
  const [mediaMessage, setMediaMessage] = useState('')
  const [mediaPreviewUrl, setMediaPreviewUrl] = useState('')
  const [mediaRefUrl, setMediaRefUrl] = useState('')  // 参考图URL（图生图）
  const [mediaStyleKey, setMediaStyleKey] = useState('')
  // 遗留项②：显式记住上次追加的风格后缀确切文本，剥离时直接 replace 该串
  const [styleSuffixText, setStyleSuffixText] = useState('')
  const [editorOpen, setEditorOpen] = useState(false)
  const [editorExporting, setEditorExporting] = useState(false)
  // 批次5a: 危险删除红色确认弹窗——待删除资产(null=未弹) + 删除请求进行中标志
  const [deleteTarget, setDeleteTarget] = useState<CoursewareAsset | null>(null)
  const [deleting, setDeleting] = useState(false)

  // ==================== 共享effect ====================
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

  // 切页或切 Tab 时清空生成框 + 参考图 + 风格选中(防串页残留)；建议列表的清空在子面板内
  useEffect(() => {
    setMediaGenPrompt(''); setMediaRefUrl(''); setMediaStyleKey('')
  }, [pageNum, mediaTab])

  // ==================== 共享处理函数 ====================
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

  // 批次5a: 红色弹窗「确认删除」回调——真正执行删除请求
  const handleConfirmDelete = async () => {
    if (!coursewareId || !deleteTarget || deleting) return
    setDeleting(true)
    try {
      await deleteCWAsset(coursewareId, deleteTarget.id)
      setMediaAssets(prev => prev.filter(a => a.id !== deleteTarget.id))
      setMediaMessage('✅ 已删除')
    } catch (e) {
      setMediaMessage('❌ 删除失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally {
      setDeleting(false)
      setDeleteTarget(null)
    }
  }

  // 批次5a: 按待删资产组装弹窗警告文案——已上云资产给最强警告（删云盘副本+引用断链+不可恢复）
  const buildDeleteWarning = (asset: CoursewareAsset): string => {
    const noun = asset.asset_type === 'video' ? '这个视频' : '这张图片'
    const effect = asset.asset_type === 'video' ? '视频将无法播放' : '图片将无法显示'
    if (asset.public_oss_url) {
      return '⚠️ ' + noun + '已上传云盘。\n删除将同时移除云盘副本——若课件页面中已使用该云盘链接，' + effect + '。\n此操作不可恢复，确定删除？'
    }
    return '确定删除' + noun + '？删除后不可恢复。'
  }

  // ==================== JSX ====================
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
        {/* 上半: 左右两栏 —— 左=AI配图建议(子面板自含), 右=生成图片(含风格快选) */}
        <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>

          {/* 左栏: AI 配图建议（批次5b拆出为独立面板） */}
          <MediaImageSuggestPanel
            coursewareId={coursewareId}
            pageNum={pageNum}
            active={mediaTab === 'image' && pageNum > 0}
            defaultSize={mediaSize}
            busyExternal={mediaGenerating}
            onFillPrompt={(prompt, size) => { setMediaGenPrompt(prompt); setMediaSize(size) }}
            onAssetCreated={(asset) => setMediaAssets(prev => [asset, ...prev])}
            notify={setMediaMessage}
          />

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
                      setMediaAssets(prev => [makeAsset(coursewareId, { id: res.asset_id, oss_url: res.url, file_size: res.file_size, mime_type: res.mime_type }), ...prev])
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
                  setMediaAssets(prev => [makeAsset(coursewareId, { id: res.asset_id, oss_url: res.url, generation_prompt: mediaGenPrompt }), ...prev])
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
            onClick={() => { const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/*'; inp.onchange = async (ev) => { const f = (ev.target as HTMLInputElement).files?.[0]; if (!f || !coursewareId) return; if (f.size > 5 * 1024 * 1024) { setMediaMessage('❌ 图片不能超过5MB'); return } setMediaGenerating(true); setMediaMessage(''); try { const res = await uploadCWImage(coursewareId, pageNum, f); setMediaMessage('✅ 上传成功！'); setMediaAssets(prev => [makeAsset(coursewareId, { id: res.asset_id, oss_url: res.url, file_size: res.file_size, mime_type: res.mime_type }), ...prev]) } catch (e) { setMediaMessage('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setMediaGenerating(false) } }; inp.click() }}
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

      {/* 已上传图片列表（批次5b拆出为 ImageAssetList） */}
      {(mediaAssets.filter(a => a.asset_type === 'image').length > 0 || courseware.style_anchor_asset_id) && mediaTab === 'image' && (
        <ImageAssetList
          coursewareId={coursewareId}
          pageNum={pageNum}
          assets={mediaAssets}
          courseware={courseware}
          anchorSetting={anchorSetting}
          anchorClearing={anchorClearing}
          onSetAnchor={(assetId) => onSetAnchor(assetId, setMediaMessage)}
          onClearAnchor={() => onClearAnchor(setMediaMessage)}
          onPreview={setMediaPreviewUrl}
          onPickRef={setMediaRefUrl}
          onAssetUpdated={(updated) => setMediaAssets(prev => prev.map(a => a.id === updated.id ? updated : a))}
          onDeleteRequest={setDeleteTarget}
          notify={setMediaMessage}
        />
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

      {/* 视频列表 + 编辑器入口（批次5b拆出为 VideoAssetList） */}
      {mediaTab === 'video' && pageNum > 0 && mediaAssets.filter(a => a.asset_type === 'video').length > 0 && (
        <VideoAssetList
          coursewareId={coursewareId}
          pageNum={pageNum}
          assets={mediaAssets}
          onOpenEditor={() => setEditorOpen(true)}
          onAssetUpdated={(updated) => setMediaAssets(prev => prev.map(a => a.id === updated.id ? updated : a))}
          onDeleteRequest={setDeleteTarget}
          notify={setMediaMessage}
        />
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

      {/* 批次5a: 危险删除红色确认弹窗——替代原生 window.confirm（已上云资产强警告 + 图片带缩略图） */}
      {deleteTarget && (
        <DangerConfirmModal
          title={deleteTarget.asset_type === 'video' ? '🗑 删除视频' : '🗑 删除图片'}
          message={buildDeleteWarning(deleteTarget)}
          confirmText={deleting ? '删除中...' : '确认删除'}
          busy={deleting}
          previewUrl={deleteTarget.asset_type === 'image' ? deleteTarget.oss_url : ''}
          onConfirm={handleConfirmDelete}
          onCancel={() => { if (!deleting) setDeleteTarget(null) }}
        />
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
            const newAsset = makeAsset(coursewareId, {
              id: res.asset_id, oss_url: res.url, asset_type: 'video',
              generation_prompt: file.name, file_size: res.file_size, mime_type: res.mime_type,
            })
            setMediaAssets(prev => [newAsset, ...prev])
            return { id: res.asset_id, url: res.url, label: file.name }
          }}
          onExport={async (clips, options) => {
            if (!coursewareId || editorExporting) return
            setEditorExporting(true); setMediaMessage('')
            try {
              // ===== S-V2 三步串行导出链: 拼接 → 烧录字幕(可选) → 混入配音(可选) =====
              // 每个可选步骤失败只记入failNotes降级继续，始终保留最近一步可用成片。

              // 第一步: FFmpeg 高级拼接出基础成片（必做，失败则整体失败走外层catch）
              const res = await advancedConcatCWVideos(coursewareId, clips)
              let finalAssetId = res.asset_id
              let finalUrl = res.url
              let withSubtitle = false   // 字幕烧录是否成功
              let withNarration = false  // 配音混音是否成功
              const failNotes: string[] = []

              // 第二步: 硬字幕烧录（可选）——在拼接成片上重编码烧录
              if (options?.burnSubtitle && options.subtitleId) {
                setMediaMessage('⏳ 成片已生成，正在烧录字幕（需重编码视频，约1-2分钟）...')
                try {
                  const burned = await burnInSubtitle(coursewareId, options.subtitleId, finalAssetId)
                  finalAssetId = burned.asset_id
                  finalUrl = burned.url
                  withSubtitle = true
                } catch (burnErr) {
                  failNotes.push('字幕烧录失败: ' + (burnErr instanceof Error ? burnErr.message : '未知错误'))
                }
              }

              // 第三步: TTS配音混音（可选）——在最新可用成片上按时间轴混入旁白
              if (options?.mixNarration && options.subtitleId) {
                setMediaMessage('⏳ 正在合成配音...')
                try {
                  const mix = await mixNarrationCWVideo(coursewareId, finalAssetId, options.subtitleId)
                  finalAssetId = mix.asset_id
                  finalUrl = mix.url
                  withNarration = true
                } catch (mixErr) {
                  failNotes.push('配音合成失败: ' + (mixErr instanceof Error ? mixErr.message : '未知错误'))
                }
              }

              // 组装产物标签与结果提示
              const featTags = [withSubtitle ? '含字幕' : '', withNarration ? '含配音' : ''].filter(Boolean)
              const finalPrompt = '编辑导出' + (featTags.length > 0 ? '(' + featTags.join('+') + ')' : '') + ' ' + clips.length + '个片段'
              const finalMsg = failNotes.length > 0
                ? '⚠️ 成片已导出，但 ' + failNotes.join('；') + '（已保留最近一步可用版本）'
                : '✅ 导出完成：' + finalPrompt
              setMediaMessage(finalMsg)
              setMediaAssets(prev => [makeAsset(coursewareId, {
                id: finalAssetId, oss_url: finalUrl, asset_type: 'video',
                generation_prompt: finalPrompt, mime_type: 'video/mp4',
              }), ...prev])
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
