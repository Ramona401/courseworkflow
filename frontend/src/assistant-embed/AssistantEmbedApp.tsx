/**
 * 公开学生端教学智能体应用编排组件。
 *
 * 本文件只负责：
 *   - 解析iframe运行环境；
 *   - 启动页、加载页、错误页和对话页之间的切换；
 *   - 页面标题、消息滚动和父页面高度通信。
 *
 * 会话、令牌、SSE和恢复状态由useAssistantEmbedRuntime集中管理。
 */

import {
  useEffect,
  useMemo,
  useRef,
} from 'react'

import {
  AssistantEmbedConversation,
  AssistantEmbedStandaloneMessage,
  AssistantEmbedStartScreen,
} from './AssistantEmbedUI'

import {
  assistantEmbedParentOriginFromReferrer,
  type AssistantEmbedBootstrap,
} from './assistantEmbedSupport'

import {
  useAssistantEmbedRuntime,
} from './useAssistantEmbedRuntime'

function useParentHeightBridge(
  publicId: string,
  targetOrigin: string,
) {
  useEffect(() => {
    if (
      window.parent === window ||
      !targetOrigin
    ) {
      return
    }

    const sendHeight = () => {
      const height = Math.max(
        320,
        Math.ceil(document.documentElement.scrollHeight),
        Math.ceil(document.body.scrollHeight),
      )

      window.parent.postMessage(
        {
          type: 'tedna-assistant-height',
          public_id: publicId,
          height,
        },
        targetOrigin,
      )
    }

    const observer =
      typeof ResizeObserver === 'undefined'
        ? null
        : new ResizeObserver(sendHeight)

    observer?.observe(document.documentElement)
    observer?.observe(document.body)

    window.addEventListener('load', sendHeight)
    window.addEventListener('resize', sendHeight)
    requestAnimationFrame(sendHeight)

    return () => {
      observer?.disconnect()
      window.removeEventListener('load', sendHeight)
      window.removeEventListener('resize', sendHeight)
    }
  }, [publicId, targetOrigin])
}

export default function AssistantEmbedApp({
  bootstrap,
}: {
  bootstrap: AssistantEmbedBootstrap
}) {
  const framed = window.self !== window.top

  const parentOrigin = useMemo(
    () => assistantEmbedParentOriginFromReferrer(),
    [],
  )

  const messageEndRef = useRef<HTMLDivElement>(null)

  const runtime = useAssistantEmbedRuntime({
    bootstrap,
    framed,
    parentOrigin,
  })

  useParentHeightBridge(
    bootstrap.publicId,
    parentOrigin,
  )

  useEffect(() => {
    document.title = bootstrap.title
  }, [bootstrap.title])

  useEffect(() => {
    messageEndRef.current?.scrollIntoView({
      block: 'end',
      behavior: runtime.sending
        ? 'smooth'
        : 'auto',
    })
  }, [
    runtime.messages,
    runtime.sending,
    runtime.streamingText,
  ])

  if (runtime.startupState === 'blocked') {
    return (
      <AssistantEmbedStandaloneMessage
        icon="🔒"
        title="请在老师提供的课件中使用"
        message="这个学习互动不能作为普通网页直接打开。请返回老师提供的课件或授课平台。"
      />
    )
  }

  if (
    framed &&
    !runtime.hasStarted
  ) {
    return (
      <AssistantEmbedStartScreen
        bootstrap={bootstrap}
        onStart={() => {
          void runtime.startSession()
        }}
      />
    )
  }

  if (runtime.startupState === 'loading') {
    return (
      <AssistantEmbedStandaloneMessage
        icon="✨"
        title="正在准备本页学习互动"
        message={
          runtime.recovering
            ? '正在恢复学习会话…'
            : '马上就好，请稍候…'
        }
        loading
      />
    )
  }

  if (runtime.startupState === 'error') {
    return (
      <AssistantEmbedStandaloneMessage
        icon="⚠️"
        title="暂时无法开始学习互动"
        message={
          runtime.notice?.text ||
          '请稍后重新尝试，或请老师检查当前页面的发布状态。'
        }
        actionLabel="重新尝试"
        onAction={() => {
          void runtime.startSession()
        }}
      />
    )
  }

  if (!runtime.session) {
    return (
      <AssistantEmbedStandaloneMessage
        icon="✨"
        title="正在准备学习内容"
        message="正在同步本次学习互动…"
        loading
      />
    )
  }

  return (
    <AssistantEmbedConversation
      bootstrap={bootstrap}
      session={runtime.session}
      messages={runtime.messages}
      streamingText={runtime.streamingText}
      input={runtime.input}
      sending={runtime.sending}
      canSend={runtime.canSend}
      statusText={runtime.statusText}
      notice={runtime.notice}
      messageEndRef={messageEndRef}
      onInputChange={runtime.setInput}
      onSend={runtime.sendMessage}
      onRestart={() => {
        void runtime.startSession()
      }}
    />
  )
}
