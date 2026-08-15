/**
 * ConversationAttachmentInput.tsx — 对话区多附件队列控制器
 *
 * 一次多选/连续追加；独立解析、重试、删除；成功受理后消费本轮附件；
 * 最近一次附件批次用于“重新回答”同一消息；图片可转正式教材依据链。
 */

import {
  forwardRef, useCallback, useEffect, useImperativeHandle, useRef, useState,
} from 'react'
import apiClient from '@/api/client'
import {
  CONVERSATION_ATTACHMENT_ACCEPT,
  prepareConversationAttachment,
  queuePendingTextbookImageFiles,
} from './conversationAttachmentProcessing'
import {
  CONVERSATION_ATTACHMENT_PROCESS_CONCURRENCY,
  MAX_CONVERSATION_ATTACHMENTS,
  MAX_CONVERSATION_ATTACHMENT_INJECT_RUNES,
  buildConversationAttachmentMaterial,
  conversationAttachmentFingerprint,
  createConversationAttachmentItem,
  getConversationAttachmentQueueStats,
  isConversationAttachmentImage,
  type ConversationAttachmentItem,
} from './conversationAttachmentQueue'
import ConversationAttachmentTray from './ConversationAttachmentTray'

export {
  consumePendingTextbookImageFiles,
  queuePendingTextbookImageFiles,
} from './conversationAttachmentProcessing'

export interface ConversationAttachmentInputHandle {
  openFilePicker: () => void
}

interface Props {
  planID: string
  textbookEnabled: boolean
  onBlockingChange: (blocking: boolean) => void
  onOpenTextbook: () => void
}

interface AttachmentBatchMeta {
  planID: string
  source: 'queue' | 'retry'
  ids: string[]
  material: string
  message: string
}

interface RetryBatch {
  message: string
  material: string
}

type AttachmentConfigWithMeta = {
  __tednaAttachmentBatch?: AttachmentBatchMeta
}

const ConversationAttachmentInput =
  forwardRef<ConversationAttachmentInputHandle, Props>(
    function ConversationAttachmentInput(
      { planID, textbookEnabled, onBlockingChange, onOpenTextbook },
      ref,
    ) {
      const inputRef = useRef<HTMLInputElement>(null)
      const dragDepthRef = useRef(0)
      const [dragActive, setDragActive] = useState(false)
      const [attachments, setAttachments] = useState<ConversationAttachmentItem[]>([])
      const attachmentsRef = useRef<ConversationAttachmentItem[]>([])
      const [notice, setNotice] = useState('')
      const sequenceRef = useRef(0)

      const activeProcessorsRef = useRef(0)
      const processorWaitersRef = useRef<Array<() => void>>([])
      const processingPromisesRef = useRef<Map<string, Promise<void>>>(new Map())
      const retryBatchRef = useRef<RetryBatch | null>(null)

      const commitAttachments = useCallback((
        updater:
          | ConversationAttachmentItem[]
          | ((previous: ConversationAttachmentItem[]) => ConversationAttachmentItem[]),
      ) => {
        const previous = attachmentsRef.current
        const next = typeof updater === 'function' ? updater(previous) : updater
        attachmentsRef.current = next
        setAttachments(next)
      }, [])

      const updateItem = useCallback((
        id: string,
        updater: (item: ConversationAttachmentItem) => ConversationAttachmentItem,
      ) => {
        commitAttachments(previous =>
          previous.map(item => item.id === id ? updater(item) : item),
        )
      }, [commitAttachments])

      const removeItem = useCallback((id: string) => {
        commitAttachments(previous => previous.filter(item => item.id !== id))
      }, [commitAttachments])

      const clearAll = useCallback(() => {
        commitAttachments([])
        setNotice('')
      }, [commitAttachments])

      const acquireProcessor = useCallback(() => new Promise<void>(resolve => {
        if (
          activeProcessorsRef.current <
          CONVERSATION_ATTACHMENT_PROCESS_CONCURRENCY
        ) {
          activeProcessorsRef.current += 1
          resolve()
          return
        }
        processorWaitersRef.current.push(resolve)
      }), [])

      const releaseProcessor = useCallback(() => {
        const next = processorWaitersRef.current.shift()
        if (next) {
          next()
          return
        }
        activeProcessorsRef.current = Math.max(
          0,
          activeProcessorsRef.current - 1,
        )
      }, [])

      const processItem = useCallback((item: ConversationAttachmentItem) => {
        const work = (async () => {
          await acquireProcessor()
          try {
            updateItem(item.id, current => ({
              ...current,
              status: 'processing',
              progress: '正在读取文件…',
              text: '',
              charCount: 0,
              error: '',
            }))

            const result = await prepareConversationAttachment(
              item.file,
              progress => {
                updateItem(item.id, current => ({
                  ...current,
                  status: 'processing',
                  progress,
                  error: '',
                }))
              },
            )

            updateItem(item.id, current => ({
              ...current,
              status: 'ready',
              progress: '',
              text: result.text,
              charCount: result.charCount,
              error: '',
            }))
          } catch (error) {
            updateItem(item.id, current => ({
              ...current,
              status: 'error',
              progress: '',
              text: '',
              charCount: 0,
              error: error instanceof Error
                ? error.message
                : '文件处理失败，请重试。',
            }))
          } finally {
            releaseProcessor()
          }
        })()

        processingPromisesRef.current.set(item.id, work)
        void work.finally(() => {
          if (processingPromisesRef.current.get(item.id) === work) {
            processingPromisesRef.current.delete(item.id)
          }
        })
        return work
      }, [acquireProcessor, releaseProcessor, updateItem])

      const addFiles = useCallback((files: FileList | File[]) => {
        const incoming = Array.from(files)
        if (incoming.length === 0) return

        const existing = attachmentsRef.current
        const fingerprints = new Set(existing.map(item => item.fingerprint))
        const unique = incoming.filter(file => {
          const fingerprint = conversationAttachmentFingerprint(file)
          if (fingerprints.has(fingerprint)) return false
          fingerprints.add(fingerprint)
          return true
        })

        const available = Math.max(
          0,
          MAX_CONVERSATION_ATTACHMENTS - existing.length,
        )
        const accepted = unique.slice(0, available)

        setNotice(
          accepted.length < incoming.length
            ? `一次最多保留 ${MAX_CONVERSATION_ATTACHMENTS} 个附件；重复文件或超出数量的文件没有再次加入。`
            : '',
        )
        if (accepted.length === 0) return

        const newItems = accepted.map(file => {
          sequenceRef.current += 1
          return createConversationAttachmentItem(file, sequenceRef.current)
        })

        commitAttachments(previous => [...previous, ...newItems])
        newItems.forEach(item => { void processItem(item) })
      }, [commitAttachments, processItem])

      useImperativeHandle(ref, () => ({
        openFilePicker: () => inputRef.current?.click(),
      }), [])

      const stats = getConversationAttachmentQueueStats(attachments)

      useEffect(() => {
        onBlockingChange(stats.blocking)
        return () => onBlockingChange(false)
      }, [stats.blocking, onBlockingChange])

      useEffect(() => {
        const requestID = apiClient.interceptors.request.use(async config => {
          const url = typeof config.url === 'string' ? config.url : ''
          const match = /^\/lesson-plans\/plans\/([^/]+)\/chat(?:\?|$)/.exec(url)

          if (!match || decodeURIComponent(match[1]) !== planID) {
            return config
          }

          const currentIDs = new Set(attachmentsRef.current.map(item => item.id))
          const pending = Array.from(processingPromisesRef.current.entries())
            .filter(([id]) => currentIDs.has(id))
            .map(([, promise]) => promise)

          if (pending.length > 0) {
            await Promise.allSettled(pending)
          }

          const current = attachmentsRef.current
          const currentStats = getConversationAttachmentQueueStats(current)
          if (
            currentStats.totalInjectRunes >
            MAX_CONVERSATION_ATTACHMENT_INJECT_RUNES
          ) {
            throw new Error(
              '本轮附件内容合计过长，请移除部分附件后再发送。',
            )
          }

          const data =
            config.data && typeof config.data === 'object'
              ? config.data as Record<string, unknown>
              : {}
          const message = typeof data.message === 'string' ? data.message : ''

          const material = buildConversationAttachmentMaterial(current)
          const readyIDs = current
            .filter(item => item.status === 'ready' && item.text.trim())
            .map(item => item.id)

          let useMaterial = material
          let source: 'queue' | 'retry' | null =
            readyIDs.length > 0 ? 'queue' : null

          if (
            !useMaterial &&
            retryBatchRef.current &&
            message &&
            message === retryBatchRef.current.message
          ) {
            useMaterial = retryBatchRef.current.material
            source = 'retry'
          } else if (
            !useMaterial &&
            retryBatchRef.current &&
            message !== retryBatchRef.current.message
          ) {
            retryBatchRef.current = null
          }

          if (!useMaterial || !source) return config

          const existing =
            typeof data.ref_material === 'string'
              ? data.ref_material.trim()
              : ''

          config.data = {
            ...data,
            ref_material: existing
              ? `${existing}\n\n${useMaterial}`
              : useMaterial,
          }

          const configWithMeta =
            config as typeof config & AttachmentConfigWithMeta
          configWithMeta.__tednaAttachmentBatch = {
            planID,
            source,
            ids: source === 'queue' ? readyIDs : [],
            material: useMaterial,
            message,
          }
          return config
        })

        const responseID = apiClient.interceptors.response.use(
          response => {
            const configWithMeta =
              response.config as typeof response.config & AttachmentConfigWithMeta
            const meta = configWithMeta.__tednaAttachmentBatch

            if (!meta || meta.planID !== planID) return response

            if (meta.source === 'queue') {
              retryBatchRef.current = {
                message: meta.message,
                material: meta.material,
              }
              const sent = new Set(meta.ids)
              commitAttachments(previous =>
                previous.filter(item => !sent.has(item.id)),
              )
            }
            return response
          },
          error => Promise.reject(error),
        )

        return () => {
          apiClient.interceptors.request.eject(requestID)
          apiClient.interceptors.response.eject(responseID)
        }
      }, [planID, commitAttachments])

      useEffect(() => {
        const hasFiles = (event: DragEvent) =>
          Array.from(event.dataTransfer?.types || []).includes('Files')

        const insideDedicatedFileModal = (event: DragEvent) => {
          const target = event.target
          return target instanceof Element &&
            Boolean(target.closest('[data-conversation-file-drop-scope]'))
        }

        const onDragEnter = (event: DragEvent) => {
          if (!hasFiles(event) || insideDedicatedFileModal(event)) return
          event.preventDefault()
          dragDepthRef.current += 1
          setDragActive(true)
        }

        const onDragOver = (event: DragEvent) => {
          if (!hasFiles(event) || insideDedicatedFileModal(event)) return
          event.preventDefault()
          if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
        }

        const onDragLeave = (event: DragEvent) => {
          if (!hasFiles(event) || insideDedicatedFileModal(event)) return
          dragDepthRef.current = Math.max(0, dragDepthRef.current - 1)
          if (dragDepthRef.current === 0) setDragActive(false)
        }

        const onDrop = (event: DragEvent) => {
          if (!hasFiles(event) || insideDedicatedFileModal(event)) return
          event.preventDefault()
          dragDepthRef.current = 0
          setDragActive(false)
          const files = event.dataTransfer?.files
          if (files?.length) addFiles(files)
        }

        window.addEventListener('dragenter', onDragEnter)
        window.addEventListener('dragover', onDragOver)
        window.addEventListener('dragleave', onDragLeave)
        window.addEventListener('drop', onDrop)

        return () => {
          window.removeEventListener('dragenter', onDragEnter)
          window.removeEventListener('dragover', onDragOver)
          window.removeEventListener('dragleave', onDragLeave)
          window.removeEventListener('drop', onDrop)
        }
      }, [addFiles])

      const retryItem = useCallback((item: ConversationAttachmentItem) => {
        updateItem(item.id, current => ({
          ...current,
          status: 'processing',
          progress: '等待重试…',
          error: '',
        }))
        void processItem(item)
      }, [processItem, updateItem])

      const promoteToTextbook = useCallback((
        item: ConversationAttachmentItem,
      ) => {
        if (!isConversationAttachmentImage(item.file)) return
        queuePendingTextbookImageFiles([item.file])
        removeItem(item.id)
        onOpenTextbook()
      }, [onOpenTextbook, removeItem])

      return (
        <>
          <input
            ref={inputRef}
            type="file"
            multiple
            accept={CONVERSATION_ATTACHMENT_ACCEPT}
            onChange={event => {
              if (event.target.files?.length) addFiles(event.target.files)
              event.target.value = ''
            }}
            style={{ display: 'none' }}
          />

          <ConversationAttachmentTray
            attachments={attachments}
            stats={stats}
            notice={notice}
            textbookEnabled={textbookEnabled}
            dragActive={dragActive}
            onRemove={removeItem}
            onRetry={retryItem}
            onPromoteToTextbook={promoteToTextbook}
            onClearAll={clearAll}
          />
        </>
      )
    },
  )

export default ConversationAttachmentInput
