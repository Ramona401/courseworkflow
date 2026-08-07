/**
 * 应用入口文件。
 *
 * 职责：
 *   - 引入全局样式；
 *   - 启动前端新版本自动检测；
 *   - 渲染App根组件。
 */
import {
  StrictMode,
} from 'react'
import {
  createRoot,
} from 'react-dom/client'

import './index.css'

import App from './App'

import {
  startAppVersionRefresh,
} from './appVersionRefresh'

/**
 * 检测器只在生产构建中生效。
 *
 * 本次功能上线后，当前已经打开的旧页面需要最后手动刷新一次；
 * 加载本检测器后，未来部署会自动刷新到新版本。
 */
startAppVersionRefresh()

createRoot(
  document.getElementById(
    'root',
  )!,
).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
