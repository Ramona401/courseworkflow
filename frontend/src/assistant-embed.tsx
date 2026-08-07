/**
 * 教学智能体公开iframe学生端Vite入口。
 *
 * 本文件只完成安全壳启动数据解析和React挂载。
 * 学生端应用编排位于assistant-embed/AssistantEmbedApp.tsx，
 * 会话运行状态位于assistant-embed/useAssistantEmbedRuntime.ts。
 */

import {
  createRoot,
} from 'react-dom/client'

import AssistantEmbedApp from './assistant-embed/AssistantEmbedApp'

import {
  readAssistantEmbedBootstrap,
} from './assistant-embed/assistantEmbedSupport'

const rootElement =
  document.getElementById(
    'assistant-embed-root',
  )

if (rootElement) {
  const bootstrap =
    readAssistantEmbedBootstrap(
      rootElement,
    )

  if (bootstrap) {
    createRoot(rootElement).render(
      <AssistantEmbedApp
        bootstrap={bootstrap}
      />,
    )
  } else {
    rootElement.innerHTML = ''

    const message =
      document.createElement(
        'main',
      )

    message.className =
      'embed-fallback'

    message.textContent =
      '学习互动配置无效，请联系授课老师。'

    rootElement.appendChild(
      message,
    )
  }
}
