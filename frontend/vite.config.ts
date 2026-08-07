import {
  defineConfig,
  type Plugin,
} from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

/**
 * 每次正式构建生成唯一前端版本号。
 *
 * 版本号同时写入：
 *   1. 构建后的JavaScript常量；
 *   2. dist/version.json。
 *
 * 已打开的旧页面通过轮询version.json发现新版本，
 * 再使用带版本查询参数的地址重新加载，绕过旧index.html缓存。
 */
const frontendBuildID =
  process.env.VITE_TEDNA_BUILD_ID?.trim() ||
  `${Date.now().toString(36)}-${process.pid}`

const frontendBuiltAt =
  new Date().toISOString()

/**
 * 输出不带内容哈希的轻量版本清单。
 *
 * 浏览器请求时会同时使用：
 *   - cache: no-store；
 *   - 当前时间查询参数。
 *
 * 因此不依赖特定Nginx缓存配置，也不会影响带哈希JS/CSS的长期缓存。
 */
function tednaBuildManifestPlugin(): Plugin {
  return {
    name: 'tedna-build-manifest',
    apply: 'build',

    generateBundle() {
      this.emitFile({
        type: 'asset',
        fileName: 'version.json',
        source: JSON.stringify(
          {
            build_id: frontendBuildID,
            built_at: frontendBuiltAt,
          },
          null,
          2,
        ),
      })
    },
  }
}

export default defineConfig({
  plugins: [
    react(),
    tednaBuildManifestPlugin(),
  ],

  /**
   * 把当前构建ID冻结进教师端主应用。
   *
   * 旧页面读取服务器version.json后，
   * 可与自己加载时的构建ID做稳定比较。
   */
  define: {
    'import.meta.env.VITE_TEDNA_BUILD_ID':
      JSON.stringify(frontendBuildID),
  },

  resolve: {
    alias: {
      '@': path.resolve(
        __dirname,
        './src',
      ),
    },
  },

  build: {
    rollupOptions: {
      /**
       * 双入口：
       *   - main：现有教师端SPA；
       *   - assistant-embed：Go动态安全壳加载的独立免登录学生端模块。
       *
       * 学生端入口固定输出/assets/assistant-embed.js，便于Go HTML安全壳引用；
       * 其余入口和共享chunk继续使用内容哈希，保留长期缓存能力。
       */
      input: {
        main: path.resolve(
          __dirname,
          'index.html',
        ),
        'assistant-embed': path.resolve(
          __dirname,
          'src/assistant-embed.tsx',
        ),
      },

      output: {
        entryFileNames(chunkInfo) {
          if (
            chunkInfo.name ===
            'assistant-embed'
          ) {
            return 'assets/assistant-embed.js'
          }

          return 'assets/[name]-[hash].js'
        },

        chunkFileNames:
          'assets/[name]-[hash].js',
        assetFileNames:
          'assets/[name]-[hash][extname]',

        /**
         * v142分包优化：
         *   1. vendor-react：React生态核心库；
         *   2. vendor-docx：Word导出库；
         *   3. vendor-axios：教师端HTTP客户端。
         *
         * 独立学生端使用原生fetch，不会引入axios和教师JWT客户端。
         */
        manualChunks(id) {
          if (
            id.includes(
              'node_modules/react/',
            ) ||
            id.includes(
              'node_modules/react-dom/',
            ) ||
            id.includes(
              'node_modules/react-router',
            ) ||
            id.includes(
              'node_modules/scheduler/',
            )
          ) {
            return 'vendor-react'
          }

          if (
            id.includes(
              'node_modules/docx/',
            ) ||
            id.includes(
              'node_modules/file-saver/',
            )
          ) {
            return 'vendor-docx'
          }

          if (
            id.includes(
              'node_modules/axios/',
            )
          ) {
            return 'vendor-axios'
          }
        },
      },
    },
  },

  server: {
    proxy: {
      '/api': {
        target:
          'http://127.0.0.1:8080',
        changeOrigin: true,
      },

      /**
       * 本地开发时让/embed/assistant/{public_id}也由Go动态返回，
       * 确保开发环境与生产环境使用同一套CSP和公开描述逻辑。
       */
      '/embed/assistant': {
        target:
          'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
})
