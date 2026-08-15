/**
 * ConversationAttachmentTray.tsx — composer多附件卡片与拖拽遮罩
 */

import { C } from '../components/workshopConstants'
import {
  MAX_CONVERSATION_ATTACHMENTS,
  isConversationAttachmentImage,
  type ConversationAttachmentItem,
  type ConversationAttachmentQueueStats,
} from './conversationAttachmentQueue'

interface Props {
  attachments: ConversationAttachmentItem[]
  stats: ConversationAttachmentQueueStats
  notice: string
  textbookEnabled: boolean
  dragActive: boolean
  onRemove: (id: string) => void
  onRetry: (item: ConversationAttachmentItem) => void
  onPromoteToTextbook: (item: ConversationAttachmentItem) => void
  onClearAll: () => void
}

export default function ConversationAttachmentTray({
  attachments,
  stats,
  notice,
  textbookEnabled,
  dragActive,
  onRemove,
  onRetry,
  onPromoteToTextbook,
  onClearAll,
}: Props) {
  const queueTone = stats.errorCount > 0 ? '#B45309' : C.textMuted

  return (
    <>
      {attachments.length > 0 && (
        <div style={{
          margin: '0 18px 8px', padding: '9px 10px', borderRadius: '12px',
          border: `1px solid ${C.border}`, background: '#F8FAFC',
        }}>
          <div style={{
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            gap: '10px', marginBottom: '7px',
          }}>
            <div style={{ minWidth: 0, color: queueTone, fontSize: '10px', lineHeight: 1.5 }}>
              已添加 {stats.totalCount}/{MAX_CONVERSATION_ATTACHMENTS} 个附件
              {stats.processingCount > 0 ? ` · ${stats.processingCount} 个处理中` : ''}
              {stats.errorCount > 0 ? ` · ${stats.errorCount} 个未加入` : ''}
            </div>
            <button type="button" onClick={onClearAll} style={{
              flexShrink: 0, border: 'none', background: 'transparent',
              color: C.textMuted, cursor: 'pointer', fontSize: '10px',
            }}>
              清空全部
            </button>
          </div>

          {(notice || stats.blockingReason) && (
            <div style={{
              marginBottom: '7px', padding: '6px 8px', borderRadius: '7px',
              background: stats.blockingReason ? '#FFF7ED' : '#EFF6FF',
              color: stats.blockingReason ? '#9A3412' : C.primary,
              fontSize: '10px', lineHeight: 1.5,
            }}>
              {stats.blockingReason || notice}
            </div>
          )}

          <div style={{ display: 'flex', gap: '8px', overflowX: 'auto', paddingBottom: '2px' }}>
            {attachments.map(item => (
              <div key={item.id} style={{
                width: '210px', minWidth: '210px', padding: '9px 9px 8px',
                borderRadius: '10px',
                border: `1px solid ${item.status === 'error' ? '#FED7AA' : C.border}`,
                background: item.status === 'error' ? '#FFF7ED' : '#fff',
              }}>
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: '7px' }}>
                  <span style={{ flexShrink: 0, fontSize: '18px' }}>
                    {isConversationAttachmentImage(item.file) ? '🖼️' : '📄'}
                  </span>
                  <div style={{ minWidth: 0, flex: 1 }}>
                    <div title={item.fileName} style={{
                      color: C.text, fontSize: '11px', fontWeight: 600,
                      overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                    }}>
                      {item.fileName}
                    </div>
                    <div style={{
                      marginTop: '2px', minHeight: '30px',
                      color: item.status === 'error' ? '#B45309' : C.textMuted,
                      fontSize: '9px', lineHeight: 1.5, display: '-webkit-box',
                      WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden',
                    }}>
                      {item.status === 'processing'
                        ? item.progress || '正在处理…'
                        : item.status === 'ready'
                          ? `✓ 已读取 · ${item.charCount.toLocaleString()}字`
                          : `未加入本轮 · ${item.error}`}
                    </div>
                  </div>
                  <button type="button" onClick={() => onRemove(item.id)} title="移除附件" style={{
                    width: '20px', height: '20px', flexShrink: 0, border: 'none',
                    background: 'transparent', color: C.textMuted, cursor: 'pointer',
                    fontSize: '14px',
                  }}>
                    ×
                  </button>
                </div>

                <div style={{
                  marginTop: '5px', paddingLeft: '25px', display: 'flex',
                  alignItems: 'center', gap: '6px', flexWrap: 'wrap',
                }}>
                  {item.status === 'error' && (
                    <button type="button" onClick={() => onRetry(item)} style={{
                      border: 'none', background: 'transparent', color: C.primary,
                      cursor: 'pointer', fontSize: '9px', padding: 0,
                    }}>
                      重试
                    </button>
                  )}

                  {item.status === 'ready' &&
                    isConversationAttachmentImage(item.file) &&
                    textbookEnabled && (
                    <button type="button" onClick={() => onPromoteToTextbook(item)} style={{
                      padding: '2px 7px', borderRadius: '999px',
                      border: '1px solid #D1D5DB', background: '#fff',
                      color: C.textSec, cursor: 'pointer', fontSize: '9px',
                    }}>
                      📚 作为教材依据
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {dragActive && (
        <div style={{
          position: 'fixed', inset: 0, zIndex: 12000, display: 'flex',
          alignItems: 'center', justifyContent: 'center', pointerEvents: 'none',
          background: 'rgba(79,123,232,0.12)', backdropFilter: 'blur(2px)',
        }}>
          <div style={{
            padding: '22px 34px', borderRadius: '18px',
            border: `2px dashed ${C.primary}`, background: '#fff', color: C.primary,
            boxShadow: '0 18px 48px rgba(0,0,0,0.16)',
            fontSize: '16px', fontWeight: 700, textAlign: 'center',
          }}>
            📎 松开即可添加到本轮对话
            <div style={{
              marginTop: '6px', color: C.textMuted, fontSize: '11px', fontWeight: 400,
            }}>
              可一次添加多个 · 最多 {MAX_CONVERSATION_ATTACHMENTS} 个
            </div>
          </div>
        </div>
      )}
    </>
  )
}
