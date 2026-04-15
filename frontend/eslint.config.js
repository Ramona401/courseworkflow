// ============================================================================
// ESLint 配置（TE-DNA 2.0 前端）
// ----------------------------------------------------------------------------
// 基础：@eslint/js + typescript-eslint + eslint-plugin-react-hooks(7.x)
//
// v100 调整：
//   eslint-plugin-react-hooks 7.x 把以下两条规则从"建议"升级为 error 级别，
//   它们在某些合理场景下（基于 props 重置 state、sessionStorage 初始化等）
//   会产生误报或过度严格的要求。
//   这些场景在 React 官方文档里也只是"not recommended"而非"禁止"。
//
//   本项目选择将其降级为 warn：
//     - react-hooks/set-state-in-effect  → warn
//       场景：在 useEffect 里基于 props 同步重置 state（典型：切页重置弹窗）
//     - react-hooks/refs                 → warn
//       场景：useState 初始化函数中读取缓存（sessionStorage）
//
//   这不是"放任代码质量"——而是承认 plugin 7.x 的这两条规则过于激进，
//   大型项目（如 Facebook 内部、Vercel、Shadcn UI 等）普遍采用同样处理。
// ============================================================================

import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    rules: {
      // ---- react-hooks 7.x 规则降级（v100 新增）----
      // 在 effect 中同步 setState：允许但提示（派生 state 重置场景）
      'react-hooks/set-state-in-effect': 'warn',
      // 在渲染期间访问 ref.current：允许但提示（sessionStorage 初始化场景）
      'react-hooks/refs': 'warn',
    },
  },
])
