/**
 * LessonPlanWordImportPreviewCard.tsx
 *
 * 展示后端安全解析后的浏览器安全预览：
 *   - 原文件基本信息；
 *   - 表格、单元格、图片、公式和可编辑块指标；
 *   - 复杂OOXML对象的解析告警；
 *   - 供AI评审使用的语义Markdown结构预览。
 *
 * 此组件不接收storage_key、文件哈希、物理路径或原始OOXML。
 */

import type {
  LessonPlanWordImportPreview,
  LessonPlanWordPreviewWarning,
} from '@/api/lesson-plan-word-import'
import { renderMarkdown } from './workshopConstants'
import { C } from './workshopConstants'

interface Props {
  preview: LessonPlanWordImportPreview
}

function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return '未知大小'
  }

  if (bytes < 1024 * 1024) {
    return `${Math.max(
      1,
      Math.round(bytes / 1024),
    )} KB`
  }

  return `${(
    bytes /
    1024 /
    1024
  ).toFixed(2)} MB`
}

function formatExpiresAt(value: string): string {
  const date = new Date(value)

  if (Number.isNaN(date.getTime())) {
    return '24小时内'
  }

  return date.toLocaleString('zh-CN', {
    hour12: false,
  })
}

function getWarningText(
  warning:
    | LessonPlanWordPreviewWarning
    | string,
): string {
  if (typeof warning === 'string') {
    return warning
  }

  const directText = [
    warning.message,
    warning.detail,
    warning.code,
  ].find(value =>
    typeof value === 'string' &&
    value.trim().length > 0,
  )

  if (typeof directText === 'string') {
    return directText.trim()
  }

  try {
    return JSON.stringify(warning)
  } catch {
    return '文档包含暂未完全支持的复杂对象'
  }
}

export default function LessonPlanWordImportPreviewCard({
  preview,
}: Props) {
  const metrics = preview.metrics || {}

  const metricItems = [
    {
      label: '表格',
      value: metrics.table_count || 0,
    },
    {
      label: '单元格',
      value: metrics.cell_count || 0,
    },
    {
      label: '内容块',
      value: metrics.block_count || 0,
    },
    {
      label: '可编辑块',
      value:
        metrics.editable_block_count || 0,
    },
    {
      label: '图片',
      value: metrics.image_count || 0,
    },
    {
      label: '公式',
      value: metrics.formula_count || 0,
    },
  ]

  const warnings = Array.isArray(
    preview.warnings,
  )
    ? preview.warnings
    : []

  return (
    <div style={{
      marginTop: '10px',
      border: `1px solid ${C.border}`,
      borderRadius: '12px',
      overflow: 'hidden',
      background: '#FFFFFF',
    }}>
      <div style={{
        padding: '12px 14px',
        background: '#F0FDF4',
        borderBottom:
          '1px solid rgba(16,185,129,0.16)',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'flex-start',
        gap: '12px',
      }}>
        <div style={{ minWidth: 0 }}>
          <div style={{
            fontSize: '13px',
            fontWeight: 700,
            color: '#166534',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}>
            ✅ 原Word结构已安全解析
          </div>

          <div style={{
            marginTop: '3px',
            fontSize: '11px',
            color: '#4B5563',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}>
            {preview.original_file_name}
            {' · '}
            {formatFileSize(preview.file_size)}
          </div>
        </div>

        <span style={{
          flexShrink: 0,
          padding: '3px 8px',
          borderRadius: '999px',
          background: preview.can_confirm
            ? '#DCFCE7'
            : '#FEF3C7',
          color: preview.can_confirm
            ? '#166534'
            : '#92400E',
          fontSize: '10px',
          fontWeight: 700,
        }}>
          {preview.can_confirm
            ? '可以导入'
            : '暂不能确认'}
        </span>
      </div>

      <div style={{
        padding: '12px 14px',
        borderBottom: `1px solid ${C.border}`,
      }}>
        <div style={{
          display: 'grid',
          gridTemplateColumns:
            'repeat(3, minmax(0, 1fr))',
          gap: '7px',
        }}>
          {metricItems.map(item => (
            <div
              key={item.label}
              style={{
                padding: '8px 7px',
                borderRadius: '8px',
                background: '#F8FAFC',
                textAlign: 'center',
              }}
            >
              <div style={{
                color: C.primary,
                fontSize: '16px',
                fontWeight: 800,
              }}>
                {item.value}
              </div>

              <div style={{
                marginTop: '2px',
                color: C.textSec,
                fontSize: '10px',
              }}>
                {item.label}
              </div>
            </div>
          ))}
        </div>

        <div style={{
          marginTop: '9px',
          color: C.textMuted,
          fontSize: '10px',
          lineHeight: 1.6,
        }}>
          预解析会话有效至
          {' '}
          {formatExpiresAt(preview.expires_at)}
          。正式导入后，原DOCX会保存为私有不可变版本。
        </div>
      </div>

      {warnings.length > 0 && (
        <div style={{
          padding: '10px 14px',
          background: '#FFFBEB',
          borderBottom:
            '1px solid rgba(245,158,11,0.18)',
        }}>
          <div style={{
            color: '#92400E',
            fontSize: '11px',
            fontWeight: 700,
          }}>
            ⚠️ 解析提示
          </div>

          <div style={{
            marginTop: '5px',
            display: 'flex',
            flexDirection: 'column',
            gap: '4px',
          }}>
            {warnings
              .slice(0, 6)
              .map((warning, index) => (
                <div
                  key={`${index}-${getWarningText(
                    warning,
                  )}`}
                  style={{
                    color: '#92400E',
                    fontSize: '10px',
                    lineHeight: 1.55,
                  }}
                >
                  • {getWarningText(warning)}
                </div>
              ))}

            {warnings.length > 6 && (
              <div style={{
                color: '#B45309',
                fontSize: '10px',
              }}>
                另有
                {' '}
                {warnings.length - 6}
                {' '}
                条提示，导入后仍会保留原始DOCX。
              </div>
            )}
          </div>
        </div>
      )}

      <div style={{
        padding: '10px 14px 12px',
      }}>
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          gap: '8px',
          marginBottom: '7px',
        }}>
          <span style={{
            color: C.text,
            fontSize: '11px',
            fontWeight: 700,
          }}>
            表格与内容结构预览
          </span>

          <span style={{
            color: C.textMuted,
            fontSize: '10px',
          }}>
            复杂公式或浮动对象可能显示占位符
          </span>
        </div>

        <div style={{
          maxHeight: '260px',
          overflowY: 'auto',
          padding: '11px 12px',
          borderRadius: '8px',
          border: `1px solid ${C.border}`,
          background: '#FAFAFA',
          fontSize: '12px',
          lineHeight: 1.7,
        }}>
          {renderMarkdown(
            preview.semantic_markdown,
          )}
        </div>
      </div>
    </div>
  )
}
