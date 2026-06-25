/**
 * DangerConfirmModal.tsx — 危险操作红色确认弹窗（课件工坊共享组件，批次5a）
 *
 * 替代浏览器原生 window.confirm 的正式确认弹窗，用于"删除已上传资产"等
 * 不可恢复的危险操作（backlog 第3项）。原生 confirm 的问题：样式不可控、
 * 无法突出风险等级、长警告文案可读性差。
 *
 * 设计要点：
 *   - 红色警示视觉：顶部红色标题条 + ⚠️ 图标 + 红色实心确认按钮；
 *   - message 支持多行（按 \n 拆分渲染）；
 *   - 可选 previewUrl 显示被删资产缩略图，让老师"看着图确认"，减少误删；
 *   - busy=true 时双按钮禁用（删除请求进行中防重复提交/误关闭）；
 *   - 点击遮罩 = 取消（busy 时不响应）；确认/取消逻辑全部由父组件回调控制。
 *
 * 当前调用方：MediaManagerPanel（图片/视频资产删除）。
 * 后续可复用：BackgroundPicker 个人图集归档删除等其它危险操作。
 */

interface Props {
  /** 弹窗标题（如 "🗑 删除图片"） */
  title: string
  /** 警告正文，支持 \n 多行 */
  message: string
  /** 确认按钮文案（如 "确认删除" / "删除中..."） */
  confirmText: string
  /** 操作进行中：禁用全部交互，防重复提交 */
  busy?: boolean
  /** 可选：被操作资产的缩略图 URL（图片资产传入，视频可不传） */
  previewUrl?: string
  /** 点击红色确认按钮 */
  onConfirm: () => void
  /** 点击取消按钮 / 点击遮罩 */
  onCancel: () => void
}

export default function DangerConfirmModal({ title, message, confirmText, busy, previewUrl, onConfirm, onCancel }: Props) {
  return (
    /* 遮罩层：z-index 高于面板内大图预览(99990)，确保弹窗永远在最上层 */
    <div
      onClick={() => { if (!busy) onCancel() }}
      style={{ position: 'fixed', inset: 0, zIndex: 99992, background: 'rgba(0,0,0,0.55)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 20 }}
    >
      {/* 弹窗主体：阻止冒泡防止点内容区误触遮罩取消 */}
      <div onClick={e => e.stopPropagation()} style={{ width: '100%', maxWidth: 440, borderRadius: 14, background: '#fff', overflow: 'hidden', boxShadow: '0 12px 48px rgba(0,0,0,0.35)' }}>

        {/* 红色警示标题条 */}
        <div style={{ padding: '14px 20px', background: 'linear-gradient(135deg, #DC2626, #EF4444)', display: 'flex', alignItems: 'center', gap: 10 }}>
          <span style={{ fontSize: 20 }}>⚠️</span>
          <span style={{ fontSize: 15, fontWeight: 700, color: '#fff' }}>{title}</span>
        </div>

        {/* 正文区：可选缩略图 + 多行警告文案 */}
        <div style={{ padding: '18px 20px' }}>
          {previewUrl && (
            <div style={{ textAlign: 'center', marginBottom: 14 }}>
              <img src={previewUrl} alt="待删除资产预览" style={{ maxWidth: 180, maxHeight: 110, objectFit: 'cover', borderRadius: 10, border: '2px solid #FCA5A5' }} />
            </div>
          )}
          {message.split('\n').map((line, i) => (
            <div key={i} style={{ fontSize: 13.5, color: '#374151', lineHeight: 1.7, marginBottom: line ? 4 : 8 }}>{line}</div>
          ))}
        </div>

        {/* 按钮区：取消（灰描边）+ 确认（红色实心） */}
        <div style={{ padding: '0 20px 18px', display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button
            onClick={onCancel}
            disabled={busy}
            style={{ padding: '9px 22px', borderRadius: 8, border: '1px solid #E5E7EB', background: '#fff', color: '#6B7280', fontSize: 14, fontWeight: 500, cursor: busy ? 'default' : 'pointer' }}
          >取消</button>
          <button
            onClick={onConfirm}
            disabled={busy}
            style={{ padding: '9px 22px', borderRadius: 8, border: 'none', background: busy ? '#FCA5A5' : 'linear-gradient(135deg, #DC2626, #EF4444)', color: '#fff', fontSize: 14, fontWeight: 600, cursor: busy ? 'default' : 'pointer', boxShadow: busy ? 'none' : '0 2px 10px rgba(220,38,38,0.35)' }}
          >{confirmText}</button>
        </div>
      </div>
    </div>
  )
}
