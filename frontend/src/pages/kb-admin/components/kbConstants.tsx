/**
 * kbConstants.tsx — 知识库压缩入库系统前端共用常量、工具函数、共享小组件
 *
 * 被 KBCurriculumPage / KBReviewPanel / KBUploadForm / KBJobList 共同依赖，故最先建立。
 * 配色 C 复用 admin 页同一套色板（见 admin/components/adminConstants.ts），保证全站视觉统一，
 * 不凭空发明颜色。
 *
 * 提供：
 *   - C                配色常量（与 admin 对齐）
 *   - fileToDataURI    单文件 → data URI（图片多模态上传必需）
 *   - filesToDataURIs  多文件批量转 data URI
 *   - Toast            右上角轻提示（3 秒自动消失）
 *   - Spinner          旋转加载指示
 *   - StatusPill       通用状态徽章（按 config 表渲染颜色+中文）
 */
import { useEffect } from 'react'

// ==================== 配色常量（与 admin/components/adminConstants.ts 对齐） ====================
export const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981',
  successLight: 'rgba(16,185,129,0.08)',
  danger: '#EF4444',
  dangerLight: 'rgba(239,68,68,0.08)',
  warning: '#F59E0B',
  warningLight: 'rgba(245,158,11,0.08)',
  purple: '#7C3AED',
  purpleLight: 'rgba(124,58,237,0.08)',
  cyan: '#0891B2',
  cyanLight: 'rgba(8,145,178,0.08)',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  bg: '#F9FAFB',
  white: '#FFFFFF',
}

// ==================== 文件 → data URI 工具（图片多模态上传） ====================

/**
 * 单个 File 读为 data URI（形如 data:image/png;base64,xxxx）。
 * 后端 KBCreateJobRequest.image_data_uris 接收这种格式做多模态识别。
 */
export function fileToDataURI(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = () => reject(new Error(`读取文件失败：${file.name}`))
    reader.readAsDataURL(file)
  })
}

/** 多个 File 批量转 data URI（顺序保持与输入一致） */
export async function filesToDataURIs(files: File[]): Promise<string[]> {
  const out: string[] = []
  for (const f of files) {
    out.push(await fileToDataURI(f))
  }
  return out
}

// ==================== 共享小组件 ====================

/** 右上角轻提示，3 秒自动消失 */
export function Toast({ message, type, onClose }: {
  message: string
  type: 'success' | 'error'
  onClose: () => void
}) {
  useEffect(() => {
    const t = setTimeout(onClose, 3000)
    return () => clearTimeout(t)
  }, [onClose])
  return (
    <div style={{
      position: 'fixed', top: '20px', right: '20px', zIndex: 2000,
      padding: '12px 20px', borderRadius: '10px',
      background: type === 'success' ? C.success : C.danger,
      color: '#fff', fontSize: '14px', fontWeight: 600,
      boxShadow: '0 4px 16px rgba(0,0,0,0.18)', maxWidth: '420px',
    }}>
      {type === 'success' ? '✅ ' : '⚠️ '}{message}
    </div>
  )
}

/** 旋转加载指示（行内小尺寸，可传 size 调整） */
export function Spinner({ size = 18 }: { size?: number }) {
  return (
    <span style={{
      display: 'inline-block',
      width: `${size}px`, height: `${size}px`,
      border: `${Math.max(2, Math.round(size / 9))}px solid ${C.border}`,
      borderTopColor: C.primary,
      borderRadius: '50%',
      animation: 'kbspin 0.8s linear infinite',
      verticalAlign: 'middle',
    }}>
      <style>{`@keyframes kbspin { to { transform: rotate(360deg); } }`}</style>
    </span>
  )
}

/**
 * 通用状态徽章：按 config 表（status → {label,color,bg}）渲染。
 * 任务状态用 KB_JOB_STATUS_CONFIG，单元审核状态用 KB_REVIEW_STATUS_CONFIG（均来自 api/kb）。
 * 未登记的状态码 fail-safe 显示原始码 + 灰底。
 */
export function StatusPill({ status, config }: {
  status: string
  config: Record<string, { label: string; color: string; bg: string }>
}) {
  const c = config[status] || { label: status || '—', color: C.textMuted, bg: C.bg }
  return (
    <span style={{
      padding: '2px 10px', borderRadius: '6px',
      fontSize: '12px', fontWeight: 600,
      color: c.color, background: c.bg,
      whiteSpace: 'nowrap',
    }}>
      {c.label}
    </span>
  )
}
