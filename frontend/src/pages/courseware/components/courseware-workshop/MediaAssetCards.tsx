/**
 * MediaAssetCards.tsx — 课件媒体资产卡片列表（批次5b从 MediaManagerPanel 拆出）
 *
 * 导出两个列表组件：
 *   - ImageAssetList：本页图片列表（含锚点置顶卡 + 设锚点/参考/快捷操作/删除）；
 *   - VideoAssetList：本页视频列表（含「打开视频编辑器」入口 + 快捷操作/删除）。
 *
 * 顺手去重：原图片卡与视频卡各写了一份完全相同的「📋复制链接/☁️云盘上传」逻辑，
 * 现内聚为本文件私有组件 CloudActions（small 参数区分两处的小号样式差异）。
 *
 * 【P2-01 体验补全】图片操作区补「复制链接 / 复制图片 / 下载」三个快捷动作：
 *   原先图片卡只有"云盘上传后复制公网链接"一条路径，老师想把图直接拿走（贴进别处、
 *   存到本地）没有趁手入口。本次新增私有组件 QuickAssetActions：
 *     · 📋 复制链接：复制该图的可访问 URL（优先 public_oss_url 公网，否则站内 oss_url）；
 *     · 🖼 复制图片：把图片二进制写进剪贴板（navigator.clipboard.write + ClipboardItem），
 *        可直接 Ctrl+V 粘进微信/文档；浏览器不支持时降级提示改用"复制链接/下载"；
 *     · ⬇ 下载：触发浏览器下载到本地（a[download]，跨域图片先 fetch 成 blob 再存，
 *        文件名按 资产类型+id 生成）。
 *   这三个动作对图片与视频都接入（视频"复制图片"不适用，仅图片卡显示）。
 *   纯前端、不调后端、不改资产状态，失败只 notify 不阻断。
 */
import { uploadAssetToOSS } from '@/api/coursewares'
import type { CoursewareAsset, CoursewareDetail } from '@/api/coursewares'
import { C } from './workshopConstants'

/** 取资产的可访问 URL：优先公网(public_oss_url)，否则站内本地路径(oss_url) */
function assetAccessibleURL(asset: CoursewareAsset): string {
  return asset.public_oss_url || asset.oss_url || ''
}

/** 从 URL 推断下载文件名的扩展名（取最后一段路径的扩展名，失败按类型兜底） */
function guessExtFromURL(url: string, assetType: string): string {
  try {
    const path = url.split('?')[0]
    const seg = path.substring(path.lastIndexOf('/') + 1)
    const dot = seg.lastIndexOf('.')
    if (dot > 0 && dot < seg.length - 1) return seg.substring(dot) // 含点，如 .png
  } catch { /* ignore */ }
  return assetType === 'video' ? '.mp4' : '.png'
}

/**
 * 快捷操作按钮组（P2-01）：复制链接 / 复制图片(仅图片) / 下载。
 * small 区分视频卡小号样式；isImage 控制"复制图片"是否显示（视频不显示）。
 */
function QuickAssetActions({ asset, small, isImage, notify }: {
  asset: CoursewareAsset
  small?: boolean
  isImage: boolean
  notify: (msg: string) => void
}) {
  const url = assetAccessibleURL(asset)
  const btnStyle = (color: string, bg: string) => ({
    padding: small ? '3px 8px' : '4px 8px', borderRadius: small ? 5 : 6,
    border: '1px solid ' + color, background: bg, color,
    fontSize: small ? 10 : 11, cursor: 'pointer' as const, whiteSpace: 'nowrap' as const,
  })

  // 📋 复制链接：直接复制可访问 URL（不触发上云；要公网链接请用云盘按钮）
  const handleCopyLink = async () => {
    if (!url) { notify('❌ 该资产暂无可复制的链接'); return }
    try {
      await navigator.clipboard.writeText(url)
      notify('📋 图片链接已复制到剪贴板')
    } catch { notify('❌ 复制失败，请手动复制') }
  }

  // 🖼 复制图片：把图片二进制写进剪贴板，可直接粘进聊天/文档
  const handleCopyImage = async () => {
    if (!url) { notify('❌ 该图片暂无可访问地址'); return }
    // 浏览器能力检测：需 clipboard.write + ClipboardItem
    if (typeof ClipboardItem === 'undefined' || !navigator.clipboard?.write) {
      notify('⚠️ 当前浏览器不支持"复制图片"，请改用"复制链接"或"下载"')
      return
    }
    notify('⏳ 正在复制图片...')
    try {
      const resp = await fetch(url, { mode: 'cors' })
      if (!resp.ok) throw new Error('图片获取失败')
      let blob = await resp.blob()
      // 剪贴板对图片类型较挑剔：非 png 的尽量转 png 以提高兼容性（用 canvas 重绘）
      if (blob.type !== 'image/png') {
        const conv = await blobToPng(blob)
        if (conv) blob = conv
      }
      await navigator.clipboard.write([new ClipboardItem({ [blob.type]: blob })])
      notify('🖼 图片已复制，可直接 Ctrl+V 粘贴')
    } catch {
      notify('⚠️ 复制图片失败（可能是跨域限制），请改用"复制链接"或"下载"')
    }
  }

  // ⬇ 下载：跨域图片先 fetch 成 blob 再用 a[download] 存本地，文件名按类型+id 生成
  const handleDownload = async () => {
    if (!url) { notify('❌ 该资产暂无可下载地址'); return }
    notify('⏳ 正在准备下载...')
    try {
      const resp = await fetch(url, { mode: 'cors' })
      if (!resp.ok) throw new Error('资产获取失败')
      const blob = await resp.blob()
      const objURL = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = objURL
      a.download = (isImage ? 'courseware_image_' : 'courseware_video_') + asset.id + guessExtFromURL(url, asset.asset_type)
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      setTimeout(() => URL.revokeObjectURL(objURL), 1500)
      notify('⬇ 已开始下载到本地')
    } catch {
      // 跨域 fetch 被拦时退化为直接打开（浏览器多半会另存或新窗打开）
      try {
        window.open(url, '_blank')
        notify('⬇ 已在新窗口打开，可右键另存为')
      } catch { notify('❌ 下载失败，请用"复制链接"在新标签打开后另存') }
    }
  }

  return (
    <>
      <button onClick={handleCopyLink} style={btnStyle('#2563EB', 'rgba(37,99,235,0.06)')} title="复制图片的可访问链接">📋 复制链接</button>
      {isImage && (
        <button onClick={handleCopyImage} style={btnStyle('#7C3AED', 'rgba(124,58,237,0.06)')} title="把图片复制到剪贴板，可直接粘贴到聊天/文档">🖼 复制图片</button>
      )}
      <button onClick={handleDownload} style={btnStyle('#0F766E', 'rgba(15,118,110,0.06)')} title="下载该资产到本地">⬇ 下载</button>
    </>
  )
}

/** 将任意图片 blob 经 canvas 转为 png blob，供剪贴板复制提高兼容性；失败返 null */
function blobToPng(blob: Blob): Promise<Blob | null> {
  return new Promise((resolve) => {
    try {
      const img = new Image()
      const objURL = URL.createObjectURL(blob)
      img.onload = () => {
        try {
          const canvas = document.createElement('canvas')
          canvas.width = img.naturalWidth || img.width
          canvas.height = img.naturalHeight || img.height
          const ctx = canvas.getContext('2d')
          if (!ctx) { URL.revokeObjectURL(objURL); resolve(null); return }
          ctx.drawImage(img, 0, 0)
          canvas.toBlob((out) => { URL.revokeObjectURL(objURL); resolve(out) }, 'image/png')
        } catch { URL.revokeObjectURL(objURL); resolve(null) }
      }
      img.onerror = () => { URL.revokeObjectURL(objURL); resolve(null) }
      img.crossOrigin = 'anonymous'
      img.src = objURL
    } catch { resolve(null) }
  })
}

/** 云盘操作按钮：已上云=复制公网链接 / 未上云=上传OSS并复制（small=视频卡小号样式） */
function CloudActions({ coursewareId, asset, small, onAssetUpdated, notify }: {
  coursewareId: string
  asset: CoursewareAsset
  small?: boolean
  onAssetUpdated: (asset: CoursewareAsset) => void
  notify: (msg: string) => void
}) {
  // 两处原样式差异：图片卡 11px/4px8px/r6，视频卡 10px/3px8px/r5
  const btnStyle = (color: string, bg: string) => ({
    padding: small ? '3px 8px' : '4px 8px', borderRadius: small ? 5 : 6,
    border: '1px solid ' + color, background: bg, color,
    fontSize: small ? 10 : 11, cursor: 'pointer' as const,
  })
  if (asset.public_oss_url) {
    return (
      <button
        onClick={async () => {
          try {
            await navigator.clipboard.writeText(asset.public_oss_url!)
            notify('📋 云盘链接已复制到剪贴板')
          } catch { notify('❌ 复制失败，请手动复制') }
        }}
        style={btnStyle('#059669', 'rgba(5,150,105,0.06)')}
        title="已上传云盘，点击复制公网链接"
      >☁️ 云盘链接</button>
    )
  }
  return (
    <button
      onClick={async () => {
        if (!coursewareId) return
        notify('⏳ 正在上传到云盘...')
        try {
          const res = await uploadAssetToOSS(coursewareId, asset.id)
          await navigator.clipboard.writeText(res.oss_public_url)
          onAssetUpdated({ ...asset, public_oss_url: res.oss_public_url })
          notify('✅ 已上传云盘，链接已复制到剪贴板')
        } catch (e) { notify('❌ 上传云盘失败: ' + (e instanceof Error ? e.message : '未知错误')) }
      }}
      style={btnStyle('#0891B2', 'rgba(8,145,178,0.06)')}
      title="上传到云盘获取公网链接（供edu平台等外部引用）"
    >☁️ 上云</button>
  )
}

/** 本页图片列表（含锚点置顶卡）——与拆分前 JSX 逐行一致，仅操作改为回调冒泡 */
export function ImageAssetList({ coursewareId, pageNum, assets, courseware, anchorSetting, anchorClearing, onSetAnchor, onClearAnchor, onPreview, onPickRef, onAssetUpdated, onDeleteRequest, notify }: {
  coursewareId: string
  pageNum: number
  assets: CoursewareAsset[]
  courseware: CoursewareDetail
  anchorSetting: string
  anchorClearing: boolean
  onSetAnchor: (assetId: string) => void
  onClearAnchor: () => void
  onPreview: (url: string) => void
  onPickRef: (url: string) => void
  onAssetUpdated: (asset: CoursewareAsset) => void
  onDeleteRequest: (asset: CoursewareAsset) => void
  notify: (msg: string) => void
}) {
  const imgCount = assets.filter(a => a.asset_type === 'image').length
  return (
    <div style={{ marginTop: 16 }}>
      <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>📎 第 {pageNum} 页的图片（{imgCount}张）</div>
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
        {/* 锚点图置顶特殊项（读 courseware 会话缓存，跨页常驻）；下方 .map 已排除锚点图本身避免重复 */}
        {courseware.style_anchor_asset_id && courseware.style_anchor_url && (
          <div key={'anchor-pinned'} style={{ width: 'calc(25% - 9px)', minWidth: 180, borderRadius: 10, border: '2px solid #F59E0B', overflow: 'hidden', background: 'rgba(245,158,11,0.04)' }}>
            <img src={courseware.style_anchor_url} alt="风格锚点" onClick={() => onPreview(courseware.style_anchor_url)} style={{ width: '100%', height: 140, objectFit: 'cover', display: 'block', cursor: 'pointer' }} title="点击查看锚点大图" />
            <div style={{ padding: '8px 10px' }}>
              <div style={{ fontSize: 11, color: '#B45309', fontWeight: 600, marginBottom: 6 }}>★ 课件风格锚点</div>
              <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                <button
                  onClick={() => { onPickRef(courseware.style_anchor_url); notify('✅ 已选锚点图为参考图，本次生成将参考锚点风格与人物') }}
                  style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, cursor: 'pointer' }}
                >参考</button>
                <button onClick={onClearAnchor} disabled={anchorClearing}
                  style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid ' + C.danger, background: 'transparent', color: C.danger, fontSize: 11, cursor: anchorClearing ? 'default' : 'pointer' }}
                >{anchorClearing ? '清除中' : '✕清除锚点'}</button>
              </div>
            </div>
          </div>
        )}
        {assets.filter(a => a.asset_type === 'image' && a.id !== courseware.style_anchor_asset_id).map(asset => (
          <div key={asset.id} style={{ width: 'calc(25% - 9px)', minWidth: 180, borderRadius: 10, border: '1px solid ' + C.border, overflow: 'hidden', background: '#fff' }}>
            <img src={asset.oss_url} alt="课件图片" onClick={() => onPreview(asset.oss_url)} style={{ width: '100%', height: 140, objectFit: 'cover', display: 'block', cursor: 'pointer' }} title="点击查看大图" />
            <div style={{ padding: '8px 10px' }}>
              <div style={{ fontSize: 11, color: C.textMuted, marginBottom: 6 }}>{asset.generation_prompt ? '🤖 AI生成' : '📤 手动上传'}{asset.public_oss_url ? ' · ☁️已上云' : ''}</div>
              <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                {courseware.style_anchor_asset_id === asset.id ? (
                  <span style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #F59E0B', background: 'rgba(245,158,11,0.12)', color: '#B45309', fontSize: 11, fontWeight: 600, whiteSpace: 'nowrap' }} title="当前风格锚点">★ 锚点</span>
                ) : (
                  <button
                    onClick={() => onSetAnchor(asset.id)}
                    disabled={!!anchorSetting}
                    title="设为风格锚点：后续配图将自动参考此图风格，保持全课件视觉统一"
                    style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #F59E0B', background: 'rgba(245,158,11,0.06)', color: '#B45309', fontSize: 11, cursor: anchorSetting ? 'default' : 'pointer', whiteSpace: 'nowrap' }}
                  >{anchorSetting === asset.id ? '⏳设置中' : '⭐设为锚点'}</button>
                )}
                <button
                  onClick={() => { onPickRef(asset.oss_url); notify('✅ 已选为参考图，AI将参考此图风格生成新图片') }}
                  style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, cursor: 'pointer' }}
                >参考</button>
                {/* P2-01: 快捷操作——复制链接 / 复制图片 / 下载 */}
                <QuickAssetActions asset={asset} isImage notify={notify} />
                <CloudActions coursewareId={coursewareId} asset={asset} onAssetUpdated={onAssetUpdated} notify={notify} />
                <button
                  onClick={() => onDeleteRequest(asset)}
                  style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #EF4444', background: 'rgba(239,68,68,0.06)', color: '#EF4444', fontSize: 11, cursor: 'pointer' }}
                >删除</button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

/** 本页视频列表（含编辑器入口）——与拆分前 JSX 逐行一致，仅操作改为回调冒泡 */
export function VideoAssetList({ coursewareId, pageNum, assets, onOpenEditor, onAssetUpdated, onDeleteRequest, notify }: {
  coursewareId: string
  pageNum: number
  assets: CoursewareAsset[]
  onOpenEditor: () => void
  onAssetUpdated: (asset: CoursewareAsset) => void
  onDeleteRequest: (asset: CoursewareAsset) => void
  notify: (msg: string) => void
}) {
  const videos = assets.filter(a => a.asset_type === 'video')
  return (
    <div style={{ marginTop: 16 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <span style={{ fontSize: 13, fontWeight: 600, color: '#7C3AED' }}>🎬 第 {pageNum} 页的视频（{videos.length}个）</span>
        <button onClick={onOpenEditor} style={{ padding: '6px 14px', borderRadius: 8, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 12, fontWeight: 600, cursor: 'pointer' }}>🎬 打开视频编辑器</button>
      </div>
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
        {videos.map(asset => (
          <div key={asset.id} style={{ width: 240, borderRadius: 10, border: '1px solid ' + C.border, overflow: 'hidden', background: '#fff' }}>
            <video src={asset.oss_url} controls style={{ width: '100%', height: 135, display: 'block', background: '#000', objectFit: 'contain' }} />
            <div style={{ padding: '8px 10px' }}>
              <div style={{ fontSize: 11, color: C.textMuted, marginBottom: 4 }}>{asset.generation_prompt ? '🤖 ' + asset.generation_prompt.slice(0,25) + (asset.generation_prompt.length > 25 ? '...' : '') : '📤 手动上传'}{asset.public_oss_url ? ' · ☁️已上云' : ''}</div>
              <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                {/* P2-01: 视频快捷操作——复制链接 / 下载（视频无"复制图片"） */}
                <QuickAssetActions asset={asset} small isImage={false} notify={notify} />
                <CloudActions coursewareId={coursewareId} asset={asset} small onAssetUpdated={onAssetUpdated} notify={notify} />
                <button onClick={() => onDeleteRequest(asset)}
                  style={{ padding: '3px 8px', borderRadius: 5, border: '1px solid #EF4444', background: 'rgba(239,68,68,0.06)', color: '#EF4444', fontSize: 10, cursor: 'pointer' }}>删除</button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
