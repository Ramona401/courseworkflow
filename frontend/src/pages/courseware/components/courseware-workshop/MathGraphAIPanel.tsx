/**
 * MathGraphAIPanel.tsx — 数学图形「AI 定制」交互面板
 * (批次A,2026-07-08;批次A+拍题出图;一键AI修复,2026-07-09)
 *
 * 职责:MathGraphModal 进入 AI 模式(模板改编 / 从零生成)后,右栏参数区替换为本面板,
 * 承载"描述/拍题 → 生成 → 预览(左侧画板) → 追改/修复"的对话式定制闭环:
 *   - 首次生成:mode 按入口决定(adapt 带模板底稿 / create 纯描述);可附题目照片
 *     (教辅拍照/试卷截图),浏览器端压缩转 data URI 随请求走多模态,AI 直接读题出图;
 *   - 追改:已有 AI 代码时,自动以"当前代码为底稿 + 新要求"走 adapt 模式再调同一接口
 *     (后端零改动,天然支持无限轮追改;追改不再带图——题意已转成代码);
 *   - 一键修复:预览执行报错(previewError 非空)时显示「🔧 让 AI 修复此错误」按钮,
 *     自动把"当前代码 + 报错信息"以 adapt 模式回喂,老师无需理解报错内容
 *     (后端提示词v2的【常见错误自查清单】负责按报错定位病灶);
 *   - 重置:清空 AI 代码与图片,adapt 入口回到模板底稿预览,create 入口回到空画板。
 *
 * 代码状态提升到 MathGraphModal(预览与融入都要用),本面板只管交互与调 API。
 * 尺寸/坐标轴/网格/标注等共用控件仍由 Modal 渲染在本面板下方,不重复实现。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/MathGraphAIPanel.tsx
 * 依赖: api/mathGraph.ts
 */
import { useState, useRef } from 'react'
import { C } from './workshopConstants'
import { generateMathGraphCode } from '@/api/mathGraph'

// ==================== 类型 ====================

interface Props {
  /** 入口模式:adapt 模板改编 / create 从零生成 */
  mode: 'adapt' | 'create'
  /** 底稿模板名称(adapt 入口传入,帮 AI 理解教学场景) */
  templateName?: string
  /** 底稿构造代码(adapt 入口 = 模板当前 buildConstruction 产出;create 入口为空串) */
  baseCode: string
  /** 画板坐标范围字符串(随入口的 boundingBox 传给后端约束坐标) */
  boundingBox: string
  /** 当前 AI 代码(状态提升在 Modal;空串 = 尚未生成) */
  code: string
  /** AI 代码更新回调(生成/追改成功后回写 Modal) */
  onCode: (code: string) => void
  /** 退出 AI 模式回调 */
  onExit: () => void
  /** 外部忙碌态(融入中,禁用本面板操作) */
  busyExternal?: boolean
  /** 预览执行报错信息(Modal 传入;非空且已有 AI 代码时显示一键修复按钮) */
  previewError?: string
}

// ==================== 批次A+:题目图片浏览器端压缩 ====================

/** 压缩目标:最长边 1600px、JPEG 质量 0.85(题目文字清晰度足够,体积普遍 <500KB) */
const IMG_MAX_DIM = 1600
const IMG_JPEG_QUALITY = 0.85
/** 原始文件体积上限 15MB(手机原图普遍 3~8MB,超出直接拒绝防卡死浏览器) */
const IMG_MAX_FILE_BYTES = 15 * 1024 * 1024

/**
 * 把题目图片文件压缩为 JPEG data URI:
 * FileReader 读 → Image 解码 → canvas 等比缩放(最长边限 IMG_MAX_DIM)→ toDataURL。
 * 统一转 JPEG(题目照片无透明需求,压缩率远高于 PNG)。
 */
function compressImageToDataURI(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    if (file.size > IMG_MAX_FILE_BYTES) {
      reject(new Error('图片超过 15MB,请换一张或先压缩'))
      return
    }
    const reader = new FileReader()
    reader.onerror = () => reject(new Error('图片读取失败'))
    reader.onload = () => {
      const img = new Image()
      img.onerror = () => reject(new Error('图片解码失败,请确认是有效的图片文件'))
      img.onload = () => {
        try {
          // 等比缩放:最长边不超过 IMG_MAX_DIM,小图不放大
          const scale = Math.min(1, IMG_MAX_DIM / Math.max(img.width, img.height))
          const w = Math.round(img.width * scale)
          const h = Math.round(img.height * scale)
          const canvas = document.createElement('canvas')
          canvas.width = w
          canvas.height = h
          const ctx = canvas.getContext('2d')
          if (!ctx) { reject(new Error('浏览器不支持图片处理')); return }
          // 白底铺垫(防 PNG 透明区转 JPEG 变黑)
          ctx.fillStyle = '#FFFFFF'
          ctx.fillRect(0, 0, w, h)
          ctx.drawImage(img, 0, 0, w, h)
          resolve(canvas.toDataURL('image/jpeg', IMG_JPEG_QUALITY))
        } catch (e) {
          reject(e instanceof Error ? e : new Error('图片压缩失败'))
        }
      }
      img.src = String(reader.result)
    }
    reader.readAsDataURL(file)
  })
}

// ==================== 组件 ====================

export default function MathGraphAIPanel({
  mode, templateName, baseCode, boundingBox, code, onCode, onExit, busyExternal, previewError,
}: Props) {
  // 描述输入(每轮生成后清空,供下一轮追改)
  const [desc, setDesc] = useState('')
  // 生成运行态
  const [loading, setLoading] = useState(false)
  // 错误信息
  const [error, setError] = useState('')
  // 已完成的生成轮数(0=未生成,≥1 后按钮文案切换为"追改")
  const [rounds, setRounds] = useState(0)
  // 批次A+:题目图片 data URI(空=未附图;仅首轮生成携带)
  const [image, setImage] = useState('')
  // 图片处理中(压缩耗时几百毫秒,给个态防连点)
  const [imgBusy, setImgBusy] = useState(false)
  // 隐藏的文件选择 input
  const fileRef = useRef<HTMLInputElement | null>(null)

  const hasCode = code.trim().length > 0
  const disabled = loading || !!busyExternal
  // 首轮且未生成时才允许附图(追改阶段题意已在代码里,不再带图)
  const canAttachImage = !hasCode
  // 一键修复条件:已有 AI 代码 + 预览确实报错了
  const canAutoFix = hasCode && !!previewError?.trim()

  // ---- 批次A+:选择题目图片 → 压缩转 data URI ----
  const handlePickImage = async (file: File | null) => {
    if (!file || disabled) return
    setImgBusy(true)
    setError('')
    try {
      const uri = await compressImageToDataURI(file)
      setImage(uri)
    } catch (e) {
      setError(e instanceof Error ? e.message : '图片处理失败')
    } finally {
      setImgBusy(false)
      // 清空 input 值,同一文件可重复选择
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  // ---- 核心调用:以给定描述发起生成/追改(生成按钮与一键修复共用) ----
  const runGenerate = async (description: string) => {
    setLoading(true)
    setError('')
    try {
      const effectiveMode = hasCode ? 'adapt' : mode
      const effectiveBase = hasCode ? code : baseCode
      const result = await generateMathGraphCode({
        mode: effectiveMode,
        description,
        base_code: effectiveMode === 'adapt' ? effectiveBase : undefined,
        template_name: templateName,
        bounding_box: boundingBox,
        // 批次A+:图片仅首轮携带(追改时题意已转成底稿代码)
        image: !hasCode && image ? image : undefined,
      })
      onCode(result.code)
      setRounds(r => r + 1)
      setDesc('')
    } catch (e) {
      setError(e instanceof Error ? e.message : '生成失败,请重试')
    } finally {
      setLoading(false)
    }
  }

  // ---- 生成/追改按钮 ----
  const handleGenerate = () => {
    const d = desc.trim()
    if (disabled) return
    if (hasCode && !d) return
    if (!hasCode && !d && !image) return
    void runGenerate(d)
  }

  // ---- 一键 AI 修复:自动把报错信息组装为修复要求回喂(老师无需理解报错) ----
  const handleAutoFix = () => {
    if (disabled || !canAutoFix) return
    const fixDesc = '这段代码在画板上执行时报错了,报错信息:「' + (previewError || '').trim()
      + '」。请对照常见错误自查清单定位并修复病灶,其余部分保持不动,输出修复后的完整代码。'
    void runGenerate(fixDesc)
  }

  // ---- 重置:清空 AI 代码与图片(adapt 回模板底稿预览 / create 回空画板),轮数归零 ----
  const handleReset = () => {
    if (disabled) return
    onCode('')
    setImage('')
    setRounds(0)
    setError('')
  }

  // 占位提示按入口区分
  const placeholder = mode === 'adapt'
    ? '描述你想要的变化,如:把河改成两条折线 / 再加一个 SAS 判定对比 / 改成折叠角 C…'
    : '描述一个新图形,如:画△ABC 和直线 l,作出关于 l 的对称三角形,标出对应边相等…'

  // 生成按钮可点条件
  const canSubmit = !disabled && (hasCode ? !!desc.trim() : (!!desc.trim() || !!image))

  return (
    <div>
      {/* 面板头卡:入口说明 + 退出 */}
      <div style={{ padding: '11px 13px', borderRadius: 13, background: 'linear-gradient(135deg, #F5F3FF, #EDE9FE)', border: '1px solid #E4DDFA', marginBottom: 13 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 13, fontWeight: 800, color: '#5B21B6' }}>
            {mode === 'adapt' ? '🔧 改编模板' : '✨ AI 描述新图形'}
          </span>
          <button
            onClick={() => { if (!disabled) onExit() }}
            style={{ marginLeft: 'auto', padding: '3px 10px', borderRadius: 999, border: '1px solid #D8CEF5', background: '#fff', color: '#6D28D9', fontSize: 11, fontWeight: 700, cursor: disabled ? 'not-allowed' : 'pointer', whiteSpace: 'nowrap' }}
          >↩ 返回模板</button>
        </div>
        <div style={{ fontSize: 11.5, color: '#7A6FA8', lineHeight: 1.6, marginTop: 5 }}>
          {mode === 'adapt'
            ? '基于「' + (templateName || '当前模板') + '」做变种:说出变化或拍题上传,AI 在保留版式与教学设计的前提下改写图形。'
            : '用自然语言描述图形,或直接上传题目照片,AI 生成可交互画板(自动配滑杆、可拖点与操作提示)。'}
        </div>
      </div>

      {/* 一键 AI 修复条(预览报错时置顶显示,比人工描述报错高效得多) */}
      {canAutoFix && !loading && (
        <div style={{ marginBottom: 10, padding: '10px 12px', borderRadius: 11, background: '#FFF7ED', border: '1.5px solid #FDBA74' }}>
          <div style={{ fontSize: 12, fontWeight: 700, color: '#9A3412', lineHeight: 1.5 }}>⚠️ 图形代码执行出错(详见左侧红条)</div>
          <button
            onClick={handleAutoFix}
            disabled={disabled}
            style={{
              width: '100%', marginTop: 8, padding: '8px 0', borderRadius: 9, border: 'none', fontSize: 12.5, fontWeight: 800,
              background: disabled ? '#FDBA74' : 'linear-gradient(135deg, #F59E0B, #EA580C)',
              color: '#fff', cursor: disabled ? 'not-allowed' : 'pointer',
              boxShadow: disabled ? 'none' : '0 4px 12px rgba(234,88,12,0.3)',
            }}
          >🔧 让 AI 修复此错误</button>
        </div>
      )}

      {/* 批次A+:拍题上传区(仅首轮可用;追改阶段隐藏) */}
      {canAttachImage && (
        <div style={{ marginBottom: 10 }}>
          <input
            ref={fileRef} type="file" accept="image/*" style={{ display: 'none' }}
            onChange={e => handlePickImage(e.target.files?.[0] || null)}
          />
          {!image ? (
            <button
              onClick={() => { if (!disabled && !imgBusy) fileRef.current?.click() }}
              disabled={disabled || imgBusy}
              style={{
                width: '100%', padding: '9px 0', borderRadius: 10, cursor: (disabled || imgBusy) ? 'not-allowed' : 'pointer',
                border: '1.5px dashed #C4B5FD', background: '#FBFAFE', color: '#6D28D9', fontSize: 12.5, fontWeight: 700,
              }}
            >{imgBusy ? '⏳ 图片处理中…' : '📷 上传题目图片(拍照/截图,可选)'}</button>
          ) : (
            <div style={{ display: 'flex', gap: 10, alignItems: 'center', padding: 8, borderRadius: 10, border: '1.5px solid #E4DDFA', background: '#FBFAFE' }}>
              {/* 缩略图预览 */}
              <img src={image} alt="题目图片" style={{ width: 64, height: 64, objectFit: 'cover', borderRadius: 8, border: '1px solid #E4DDFA', flexShrink: 0 }} />
              <div style={{ minWidth: 0, flex: 1 }}>
                <div style={{ fontSize: 12, fontWeight: 700, color: '#5B21B6' }}>📷 已附题目图片</div>
                <div style={{ fontSize: 11, color: '#7A6FA8', marginTop: 3, lineHeight: 1.5 }}>AI 将直接读题出图,下方可补充说明(如"只画第2小问")</div>
              </div>
              <button
                onClick={() => { if (!disabled) setImage('') }}
                title="移除图片"
                style={{ border: 'none', background: '#F3F0FB', width: 26, height: 26, borderRadius: 8, fontSize: 13, cursor: disabled ? 'not-allowed' : 'pointer', color: '#6D28D9', flexShrink: 0 }}
              >✕</button>
            </div>
          )}
        </div>
      )}

      {/* 描述输入 */}
      <textarea
        value={desc}
        onChange={e => setDesc(e.target.value)}
        placeholder={hasCode ? '继续追改:如 角弧再大一点 / 把标注移到右上…' : (image ? '补充说明(可选):如 只画第2小问的图 / 把动点 P 做成滑杆…' : placeholder)}
        maxLength={2000}
        rows={4}
        disabled={disabled}
        style={{ width: '100%', boxSizing: 'border-box', padding: '9px 11px', borderRadius: 10, border: '1.5px solid #E9E5F5', fontSize: 12.5, lineHeight: 1.6, outline: 'none', resize: 'vertical', fontFamily: 'inherit', background: disabled ? '#F9FAFB' : '#fff' }}
      />

      {/* 生成/追改按钮 */}
      <button
        onClick={handleGenerate}
        disabled={!canSubmit}
        style={{
          width: '100%', marginTop: 9, padding: '10px 0', borderRadius: 11, border: 'none', fontSize: 13.5, fontWeight: 800,
          background: !canSubmit ? '#C4B5FD' : 'linear-gradient(135deg, #8B5CF6, #6D28D9)',
          boxShadow: !canSubmit ? 'none' : '0 5px 15px rgba(109,40,217,0.32)',
          color: '#fff', cursor: !canSubmit ? 'not-allowed' : 'pointer',
        }}
      >
        {loading ? '⏳ AI 生成中(约半分钟)…' : hasCode ? '🔁 按新要求追改' : (image ? '📷 拍题出图' : '✨ 生成图形')}
      </button>

      {/* 状态区:错误 / 成功轮数 / 重置 */}
      {error && (
        <div style={{ marginTop: 9, padding: '8px 11px', borderRadius: 9, background: '#FEE2E2', color: '#DC2626', fontSize: 12, lineHeight: 1.5 }}>
          ❌ {error}
        </div>
      )}
      {hasCode && !loading && (
        <div style={{ marginTop: 9, padding: '8px 11px', borderRadius: 9, background: '#D1FAE5', color: '#059669', fontSize: 12, lineHeight: 1.6 }}>
          ✅ 已生成(第 {rounds} 轮),请在左侧预览试拖。不满意直接输入新要求追改;满意就点下方「融入」。
          <span
            onClick={handleReset}
            style={{ marginLeft: 6, color: '#6D28D9', fontWeight: 700, cursor: disabled ? 'not-allowed' : 'pointer', textDecoration: 'underline' }}
          >重置</span>
        </div>
      )}
      {!hasCode && !loading && !error && mode === 'adapt' && (
        <div style={{ marginTop: 9, fontSize: 11.5, color: C.textMuted, lineHeight: 1.6 }}>
          左侧当前显示的是模板底稿,生成后会替换为你的变种图形。
        </div>
      )}
    </div>
  )
}
