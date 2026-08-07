// ============================================================================
// ESLint 配置（TE-DNA 2.0 前端）
// ----------------------------------------------------------------------------
// 基础：@eslint/js + typescript-eslint + eslint-plugin-react-hooks(7.x)
//
// v100 调整：
//   react-hooks/set-state-in-effect 与 react-hooks/refs 降级为warn，
//   用于兼容基于props重置状态和缓存初始化等合理场景。
//
// 教学智能体共享控件文件：
//   CoursewareAssistantEditorShared.tsx 同时导出轻量组件和共享样式常量。
//   这些常量不会保存运行状态，也不会破坏热更新语义，因此只对该文件关闭
//   react-refresh/only-export-components，不影响其他组件文件。
// ============================================================================

import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import {
  defineConfig,
  globalIgnores,
} from 'eslint/config'

export default defineConfig([
  globalIgnores([
    'dist',
  ]),
  {
    files: [
      '**/*.{ts,tsx}',
    ],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals:
        globals.browser,
    },
    rules: {
      'react-hooks/set-state-in-effect':
        'warn',
      'react-hooks/refs':
        'warn',
    },
  },
  {
    files: [
      'src/pages/courseware/CoursewareWorkshopContent.tsx',
    ],
    rules: {
      // 原工坊主体保留“失败静默但不中断主流程”的既有空catch范式。
      // 该放宽只作用于保真复制文件，不降低全项目其它文件的no-empty标准。
      'no-empty': [
        'error',
        {
          allowEmptyCatch: true,
        },
      ],
    },
  },
  {
    files: [
      'src/pages/courseware/components/courseware-workshop/CoursewareAssistantEditorShared.tsx',
    ],
    rules: {
      'react-refresh/only-export-components':
        'off',
    },
  },
])
