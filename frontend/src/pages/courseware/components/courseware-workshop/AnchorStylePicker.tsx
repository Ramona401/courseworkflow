/**
 * AnchorStylePicker.tsx — 「插图画风」选择弹窗（自动装配前置）
 *
 * 【为什么有它】
 *   全自动装配/HTML+配图两档需要一张"定调图"统一全课件配图风格（内部叫风格锚点）。
 *   但"锚点"是实现概念，老师不懂、也不该懂。本弹窗把它翻译成老师能理解的一个动作：
 *   "这套课件的插图想要什么画风？"——选一个画风，系统在背后自动生成定调图并设为锚点，
 *   全程不出现"锚点"二字。老师只做了"选画风"这一个他完全理解的决定。
 *
 * 【触发时机】仅当老师在 Step4 选了"全自动装配 / HTML+配图"且课件尚未设锚点时，由
 *   AutoAssemblyPanel 在启动装配前弹出。纯手动档永不触发。已设过锚点则直接跳过本弹窗。
 *
 * 【流程】选画风 → generateCWImage(id,1,该画风的 anchorPrompt) 生成定调图 → setStyleAnchor 设锚点
 *   → 显示定调图缩略图（老师【必须】看到并确认满意）→ 老师「🔄 换一张」重生 或
 *   「✓ 就用这个，开始装配」确认 → onConfirmed() 通知父组件一气呵成开始装配。
 *
 * 【本轮体验修复 · 两个致命问题】
 *   问题1「生成锚点后没预览就跳了」+ 问题2「弹窗莫名消失、回到大页面重新点自动生成」——
 *   两者同一根因：旧版 onAnchorChanged 回调触发父页面 loadCourseware() → 整页进入 loading 态
 *   → 本弹窗连同 AutoAssemblyPanel 一起被卸载 → 预览界面根本来不及显示、弹窗凭空消失、
 *   AutoAssemblyPanel 重新挂载后 state 全部重置回起点。
 *   修复策略：
 *     A. onAnchorChanged 改为【携带设锚点结果】回传父级，父级做「乐观更新」(setCourseware)
 *        而非整页 loadCourseware，弹窗不再被卸载，预览界面稳定停留可看可确认。
 *     B. 样板就绪(sampleUrl 非空)后，点击遮罩不再误触取消——避免老师看预览时手滑点到弹窗
 *        外围就把弹窗关掉、锚点白设、被迫回到起点。此时只能通过明确按钮（取消/换一张/就用这个）操作。
 *
 * 【定调图提示词的关键设计 · 修复"选写实却出皮克斯 / 每张图都有人物"】
 *   定调图是拿去提取整套课件画风DNA的，它决定后续每页配图的风格基调，因此：
 *     ① 用画风的 anchorPrompt（只描述画风质感/光影/色彩，不含"角色/人物"字样），
 *        而非旧版的 desc + "友好的卡通角色" 硬后缀——旧后缀对每种画风都强塞卡通人物，
 *        导致写实档被皮克斯化、且整套课件每张图都被锚定成"必须有人物"。
 *     ② 定调图主体用中性的"教学场景通用画面（如明亮教室 / 干净的知识示意背景）"，
 *        【绝不指定人物】，让画风忠实呈现自己，主体交由各页配图时按需决定
 *        （光合作用配叶片、地理配地图、故事配人物，各页自己判断）。
 *
 * pageNumber 固定传 1：Step4 时第1页方案记录已存在；锚点为课件级，挂哪页不影响全课件生效。
 */
import { useState } from 'react'
import { generateCWImage, setStyleAnchor } from '@/api/coursewares'
import type { SetStyleAnchorResult } from '@/api/coursewares'
import { C, CW_IMG_STYLES } from './workshopConstants'

interface Props {
  coursewareId: string
  /** 交付模式文案用（区分"全自动装配"/"HTML+配图"，仅影响按钮措辞） */
  skipVideo: boolean
  /** 老师确认"就用这个画风，开始装配"——父组件据此设好锚点后一气呵成启动装配 */
  onConfirmed: () => void
  /** 关闭弹窗（取消，不装配） */
  onCancel: () => void
  /**
   * 设锚点成功后回传结果给父级做「乐观更新」(setCourseware 更新 style_anchor_* 三字段)。
   * 【关键】父级绝不能在此回调里做整页 loadCourseware——那会把本弹窗卸载掉，
   * 导致预览看不到、弹窗消失、装配面板重置。回传结果让父级用 setState 局部更新即可。
   */
  onAnchorChanged: (res: SetStyleAnchorResult) => void
}

const ANCHOR_PAGE_NUM = 1

export default function AnchorStylePicker({ coursewareId, skipVideo, onConfirmed, onCancel, onAnchorChanged }: Props) {
  // 选中的画风 key
  const [selectedKey, setSelectedKey] = useState('')
  // 生成中
  const [generating, setGenerating] = useState(false)
  // 已生成的定调图 URL（非空=样板就绪，进入"确认/换一张"态）
  const [sampleUrl, setSampleUrl] = useState('')
  const [message, setMessage] = useState('')

  // 选一个画风 → 生成定调图 → 设锚点
  const pickAndGenerate = async (key: string) => {
    if (generating) return
    const style = CW_IMG_STYLES.find(s => s.key === key)
    if (!style) return
    setSelectedKey(key)
    setSampleUrl('')
    setGenerating(true)
    setMessage(`🎨 正在生成「${style.label}」风格样板（约 10–30 秒）...`)
    try {
      // 用该画风的 anchorPrompt（纯画风描述，不含人物），主体用中性教学场景，
      //   绝不指定人物——避免整套课件被锚定成"必须有人物"，也让写实/扁平等画风忠实呈现。
      const prompt =
        style.anchorPrompt +
        '。画面主体为教学场景的通用示意画面（例如明亮整洁的教室一角、或简洁干净的知识展示背景），' +
        '重点在于确立整套课件统一的视觉风格与画面质感，不需要出现具体人物。'
      const img = await generateCWImage(coursewareId, ANCHOR_PAGE_NUM, prompt)
      // 生成成功 → 立即设为锚点（内部提取风格DNA落库）
      setMessage('🎨 样板已生成，正在确立为全课件风格...')
      const res = await setStyleAnchor(coursewareId, img.asset_id)
      // 先展示预览图，让老师【看得到、能确认】——这是本轮修复的核心：预览必须稳定停留
      setSampleUrl(img.url)
      setMessage('✅ 风格样板已就绪！满意就点「就用这个，开始装配」，想换感觉点「换一张」')
      // 回传设锚点结果给父级做乐观更新（父级 setCourseware 局部更新，绝不整页 loadCourseware，
      //   否则本弹窗会被卸载，预览白显示、装配面板重置）
      onAnchorChanged(res)
    } catch (e) {
      setMessage('❌ 生成失败：' + (e instanceof Error ? e.message : '未知错误') + '，可重试或换个画风')
      setSampleUrl('')
    } finally {
      setGenerating(false)
    }
  }

  // 遮罩点击处理：
  //   · 生成中：不允许关闭（防中途误关）
  //   · 样板已就绪(sampleUrl 非空)：不允许点遮罩关闭——避免老师看预览时手滑点到外围就把弹窗关掉、
  //     锚点白设、被迫回到起点。此时只能用明确按钮（取消/换一张/就用这个）操作。
  //   · 其余（还没生成任何样板）：点遮罩=取消，符合直觉。
  const handleOverlayClick = () => {
    if (generating) return
    if (sampleUrl) return
    onCancel()
  }

  return (
    <div
      onClick={handleOverlayClick}
      style={{ position: 'fixed', inset: 0, zIndex: 99992, background: 'rgba(0,0,0,0.55)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 20 }}
    >
      <div
        onClick={e => e.stopPropagation()}
        style={{ width: '100%', maxWidth: 560, maxHeight: '88vh', overflow: 'auto', borderRadius: 16, background: '#fff', boxShadow: '0 16px 56px rgba(0,0,0,0.35)' }}
      >
        {/* 标题条 */}
        <div style={{ padding: '18px 22px', background: 'linear-gradient(135deg, #7C3AED, #6366F1)' }}>
          <div style={{ fontSize: 17, fontWeight: 700, color: '#fff', marginBottom: 4 }}>🎨 这套课件的插图想要什么画风？</div>
          <div style={{ fontSize: 12.5, color: 'rgba(255,255,255,0.85)', lineHeight: 1.6 }}>
            选一个画风，系统会先生成一张风格样板给你确认；满意后一键开始全课件装配。具体每页画什么（人物 / 事物 / 示意图）由各页内容自动决定。
          </div>
        </div>

        <div style={{ padding: '20px 22px' }}>
          {/* 画风卡片网格（动态取自 CW_IMG_STYLES） */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))', gap: 10, marginBottom: 16 }}>
            {CW_IMG_STYLES.map(style => {
              const on = selectedKey === style.key
              return (
                <button
                  key={style.key}
                  onClick={() => pickAndGenerate(style.key)}
                  disabled={generating}
                  style={{
                    textAlign: 'left', padding: '12px 14px', borderRadius: 12, cursor: generating ? 'default' : 'pointer',
                    border: '2px solid ' + (on ? '#7C3AED' : C.border),
                    background: on ? 'rgba(124,58,237,0.06)' : '#fff',
                    opacity: generating && !on ? 0.5 : 1, transition: 'all 200ms',
                  }}
                >
                  <div style={{ fontSize: 14, fontWeight: 600, color: on ? '#7C3AED' : C.textPrimary, marginBottom: 4 }}>{style.label}</div>
                  <div style={{ fontSize: 11, color: C.textMuted, lineHeight: 1.5, overflow: 'hidden', textOverflow: 'ellipsis', display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical' }}>
                    {style.desc}
                  </div>
                </button>
              )
            })}
          </div>

          {/* 消息条 */}
          {message && (
            <div style={{ padding: '10px 14px', borderRadius: 8, marginBottom: 14, fontSize: 13,
              background: message.startsWith('❌') ? '#FEE2E2' : message.startsWith('✅') ? '#D1FAE5' : '#EFF6FF',
              color: message.startsWith('❌') ? '#DC2626' : message.startsWith('✅') ? '#059669' : '#2563EB' }}>
              {message}
            </div>
          )}

          {/* 生成中转圈 */}
          {generating && (
            <div style={{ textAlign: 'center', padding: '16px 0', color: C.textMuted, fontSize: 13 }}>
              <div style={{ fontSize: 30, marginBottom: 8 }}>🎨</div>
              正在生成风格样板，请稍候…
            </div>
          )}

          {/* 样板就绪：缩略图 + 换一张 + 开始装配（本轮修复核心：预览必须稳定展示，老师看清楚再确认） */}
          {sampleUrl && !generating && (
            <div style={{ padding: 14, borderRadius: 12, background: '#F9FAFB', border: `2px solid #7C3AED`, marginBottom: 16 }}>
              <div style={{ fontSize: 13, fontWeight: 700, color: '#7C3AED', marginBottom: 10 }}>👀 风格样板预览 · 请确认是否满意</div>
              <div style={{ display: 'flex', gap: 14, alignItems: 'center' }}>
                <img src={sampleUrl} alt="风格样板" style={{ width: 120, height: 120, objectFit: 'cover', borderRadius: 10, border: '2px solid #7C3AED', flexShrink: 0 }} />
                <div style={{ flex: 1, fontSize: 12.5, color: C.textSecondary, lineHeight: 1.7 }}>
                  全课件配图将统一这个画风。<br />
                  · 满意 → 点下方「✓ 就用这个，开始装配」<br />
                  · 想换感觉 → 点「🔄 换一张」重生，或重选上方其他画风
                </div>
              </div>
            </div>
          )}

          {/* 底部按钮区 */}
          <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', flexWrap: 'wrap' }}>
            <button
              onClick={onCancel}
              disabled={generating}
              style={{ padding: '10px 20px', borderRadius: 9, border: `1px solid ${C.border}`, background: '#fff', color: C.textSecondary, fontSize: 14, cursor: generating ? 'default' : 'pointer' }}
            >取消</button>

            {sampleUrl && !generating && (
              <>
                <button
                  onClick={() => selectedKey && pickAndGenerate(selectedKey)}
                  style={{ padding: '10px 20px', borderRadius: 9, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 14, fontWeight: 600, cursor: 'pointer' }}
                >🔄 换一张</button>
                <button
                  onClick={onConfirmed}
                  style={{ padding: '10px 24px', borderRadius: 9, border: 'none', background: 'linear-gradient(135deg, #7C3AED, #6366F1)', color: '#fff', fontSize: 14, fontWeight: 600, cursor: 'pointer', boxShadow: '0 3px 12px rgba(124,58,237,0.3)' }}
                >✓ 就用这个，开始{skipVideo ? '装配' : '装配'}</button>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
