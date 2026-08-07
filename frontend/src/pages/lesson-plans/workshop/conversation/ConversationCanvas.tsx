/**
 * ConversationCanvas.tsx — 对话模式右栏教案画布。
 *
 * 功能：
 *   - 渲染并实时同步content_markdown；
 *   - 展示产物驱动的完整度清单；
 *   - 点击缺项让对话AI补充；
 *   - 复用共享编辑器完成全文编辑、图片、版本历史；
 *   - 支持目录导航和章节AI局部修改。
 */

import { C } from '../components/workshopConstants'
import LessonDocumentEditor from '../components/LessonDocumentEditor'
import type { LessonPlanContentRestoreResponse } from '@/api/lesson-plan-versions'
import type {
  LessonPlanSectionRewriteApplyResponse,
} from '@/api/lesson-plan-section-rewrite'
import { CANVAS_CHECKLIST } from './conversationScript'

interface ConversationCanvasProps {
  /** 当前教案ID */
  planID: string
  /** 教案正文Markdown */
  content: string
  /** 当前教案版本号 */
  currentVersion: number
  /**
   * 当前老师主动发起的对话或正文生成是否忙碌。
   *
   * 导入后的后台AI评审不得传入true，
   * 否则会错误锁定正文编辑和图片移除。
   */
  busy: boolean
  /** 当前教案状态是否允许编辑 */
  canEdit: boolean
  /** 保存人工编辑后的完整正文 */
  onSaveContent: (nextContent: string) => Promise<void>
  /** 恢复历史版本后同步父页面 */
  onContentRestored: (
    result: LessonPlanContentRestoreResponse,
  ) => void
  /** AI章节修改应用成功后同步父页面 */
  onSectionRewriteApplied: (
    result: LessonPlanSectionRewriteApplyResponse,
  ) => void | Promise<void>
  /** 点击缺项时让对话AI补充 */
  onFillMissing: (label: string) => void
}

/**
 * 检测正文中各组成部分是否已具备。
 * 继续沿用与后端一致的标记词包含判断，不另造判定体系。
 */
function detectChecklist(
  content: string,
): Array<{ key: string; label: string; done: boolean }> {
  const text = content || ''

  return CANVAS_CHECKLIST.map(item => ({
    key: item.key,
    label: item.label,
    done: item.patterns.some(pattern =>
      text.includes(pattern),
    ),
  }))
}

export default function ConversationCanvas({
  planID,
  content,
  currentVersion,
  busy,
  canEdit,
  onSaveContent,
  onContentRestored,
  onSectionRewriteApplied,
  onFillMissing,
}: ConversationCanvasProps) {
  const items = detectChecklist(content)
  const doneCount =
    items.filter(item => item.done).length
  const hasContent =
    Boolean(content && content.trim())

  return (
    <div style={{
      height: '100%',
      display: 'flex',
      flexDirection: 'column',
      background: C.card,
    }}>
      <div style={{
        padding: '12px 16px',
        borderBottom: `1px solid ${C.border}`,
        flexShrink: 0,
      }}>
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: '8px',
        }}>
          <span style={{
            fontSize: '13px',
            fontWeight: 700,
            color: C.text,
          }}>
            📄 教案画布
          </span>

          <span style={{
            fontSize: '11px',
            color:
              doneCount >= items.length
                ? C.success
                : C.textMuted,
            fontWeight: 600,
          }}>
            {busy
              ? '✍️ AI正在更新…'
              : `完整度 ${doneCount}/${items.length}`}
          </span>
        </div>

        <div style={{
          height: '5px',
          background: '#F3F4F6',
          borderRadius: '3px',
          overflow: 'hidden',
          marginBottom: '8px',
        }}>
          <div style={{
            height: '100%',
            borderRadius: '3px',
            width: `${
              items.length > 0
                ? (doneCount / items.length) * 100
                : 0
            }%`,
            background:
              doneCount >= items.length
                ? C.success
                : C.primary,
            transition: 'width 500ms ease',
          }} />
        </div>

        <div style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: '6px',
        }}>
          {items.map(item => (
            <button
              key={item.key}
              type="button"
              onClick={() => {
                if (!item.done && hasContent) {
                  onFillMissing(item.label)
                }
              }}
              disabled={
                item.done ||
                !hasContent ||
                busy
              }
              title={
                item.done
                  ? `${item.label}已具备`
                  : hasContent
                    ? `点击让AI补充${item.label}`
                    : '正文生成后可点击补全缺项'
              }
              style={{
                padding: '3px 10px',
                borderRadius: '12px',
                fontSize: '11px',
                border: `1px solid ${
                  item.done
                    ? 'rgba(16,185,129,0.3)'
                    : C.border
                }`,
                background: item.done
                  ? 'rgba(16,185,129,0.08)'
                  : 'transparent',
                color: item.done
                  ? C.success
                  : C.textMuted,
                fontWeight: item.done ? 600 : 400,
                cursor:
                  item.done ||
                  !hasContent ||
                  busy
                    ? 'default'
                    : 'pointer',
                transition: 'all 150ms ease',
              }}
            >
              {item.done ? '✓' : '○'} {item.label}
            </button>
          ))}
        </div>
      </div>

      <div style={{
        flex: 1,
        minHeight: 0,
        overflow: 'hidden',
      }}>
        <LessonDocumentEditor
          content={content}
          planID={planID}
          currentVersion={currentVersion}
          disabled={busy || !canEdit}
          disabledReason={
            busy
              ? 'AI正在生成或更新正文，完成后即可编辑'
              : !canEdit
                ? '当前教案状态已锁定，不允许修改正文'
                : ''
          }
          onSave={onSaveContent}
          onRestored={onContentRestored}
          onSectionRewriteApplied={
            onSectionRewriteApplied
          }
          compact
          emptyState={(
            <div style={{
              height: '100%',
              minHeight: '260px',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              color: C.textMuted,
              textAlign: 'center',
              padding: '24px',
            }}>
              <div style={{
                fontSize: '36px',
                marginBottom: '14px',
              }}>
                📝
              </div>

              <div style={{
                fontSize: '14px',
                lineHeight: 1.8,
              }}>
                你的教案会在这里一点点长出来
                <br />
                也可以点击右上角「手动填写」直接开始
              </div>
            </div>
          )}
        />
      </div>
    </div>
  )
}
