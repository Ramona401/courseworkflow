/**
 * 开发单元25与26公开运行永久协议检查。
 *
 * 本脚本不启动浏览器、不访问网络、不生成dist，也不调用AI。
 *
 * 永久防回归范围：
 *   - parent_origin和官方embed Referer三方来源绑定；
 *   - 父页面Origin缺失时fail-closed；
 *   - postMessage精确Origin；
 *   - 运行令牌不进入浏览器持久化存储；
 *   - 动态frame-ancestors不隐式加入self；
 *   - 教师端和公开端功能开关保持安全默认；
 *   - external与teacher_preview按session_kind隔离；
 *   - 已签发external令牌受实时总闸门控制；
 *   - 学生短时令牌401后自动建立新会话；
 *   - 恢复流程不自动重发学生消息。
 */

import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDirectory =
  path.dirname(
    fileURLToPath(import.meta.url),
  )
const frontendRoot =
  path.resolve(scriptDirectory, '..')
const projectRoot =
  path.resolve(frontendRoot, '..')

function read(relativePath) {
  return fs.readFileSync(
    path.resolve(projectRoot, relativePath),
    'utf8',
  )
}

function requirePattern(
  source,
  pattern,
  description,
) {
  if (!pattern.test(source)) {
    throw new Error(
      `开发单元25/26协议检查失败：缺少${description}`,
    )
  }
}

function forbidPattern(
  source,
  pattern,
  description,
) {
  if (pattern.test(source)) {
    throw new Error(
      `开发单元25/26协议检查失败：发现禁止结构${description}`,
    )
  }
}

function sliceBetween(
  source,
  startMarker,
  endMarker,
  description,
) {
  const start = source.indexOf(startMarker)
  const end = source.indexOf(
    endMarker,
    start + startMarker.length,
  )

  if (start < 0 || end <= start) {
    throw new Error(
      `开发单元25/26协议检查失败：无法定位${description}`,
    )
  }

  return source.slice(start, end)
}

const runtimeAPI = read(
  'frontend/src/api/coursewares.assistant.runtime.ts',
)
const runtimeTypes = read(
  'frontend/src/api/coursewares.assistant.types.ts',
)
const embedEntry = read(
  'frontend/src/assistant-embed.tsx',
)
const embedSupport = read(
  'frontend/src/assistant-embed/assistantEmbedSupport.ts',
)
const backendSupport = read(
  'backend/internal/handlers/assistant_runtime_handler_support.go',
)
const backendHandler = read(
  'backend/internal/handlers/assistant_runtime_handler.go',
)
const backendModel = read(
  'backend/internal/models/assistant_runtime.go',
)
const backendConfig = read(
  'backend/internal/config/config.go',
)
const backendRuntimeRoute = read(
  'backend/internal/routes/routes_assistant_runtime.go',
)
const backendSessionService = read(
  'backend/internal/services/assistant_runtime_session_service.go',
)
const backendAuthorization = read(
  'backend/internal/services/assistant_runtime_authorization.go',
)

const startFunction = sliceBetween(
  runtimeAPI,
  'export async function startAssistantRuntimeSession',
  'export async function getAssistantRuntimeSession',
  '前端会话创建函数',
)

const heightBridge = sliceBetween(
  embedEntry,
  'function useParentHeightBridge',
  'function AssistantEmbedApp',
  '学生端高度通信函数',
)

const synchronizeFunction = sliceBetween(
  embedEntry,
  'const synchronizeSession = useCallback(',
  '/**\n   * 创建全新的运行会话。',
  '学生端会话同步函数',
)

const startSessionFunction = sliceBetween(
  embedEntry,
  'const startSession = useCallback(',
  '// 把401恢复动作指向最新的startSession闭包。',
  '学生端新建会话函数',
)

const frameAncestorsFunction = sliceBetween(
  backendSupport,
  'func assistantRuntimeFrameAncestorsCSP',
  '// writeAssistantRuntimeEmbedHTML',
  '后端frame-ancestors函数',
)

// ==================== 来源绑定 ====================

requirePattern(
  startFunction,
  /publicId:\s*string,\s*anonymousClientId:\s*string,\s*parentOrigin:\s*string,/s,
  '会话创建parentOrigin参数',
)
requirePattern(
  startFunction,
  /parent_origin:\s*normalizedParentOrigin/,
  '会话创建parent_origin JSON字段',
)
requirePattern(
  startFunction,
  /referrerPolicy:\s*'same-origin'/,
  '会话创建same-origin Referer策略',
)
forbidPattern(
  startFunction,
  /referrerPolicy:\s*'no-referrer'/,
  '会话创建no-referrer策略',
)

requirePattern(
  runtimeTypes,
  /interface\s+AssistantRuntimeStartRequest[\s\S]*parent_origin:\s*string/,
  '公开运行请求parent_origin类型',
)
requirePattern(
  embedSupport,
  /document\.referrer\.trim\(\)/,
  '父页面Origin从document.referrer读取',
)
requirePattern(
  embedSupport,
  /return\s+parsed\.origin/,
  '父页面来源收敛为精确Origin',
)
requirePattern(
  embedEntry,
  /if\s*\(\s*!parentOrigin\s*\)/,
  '父页面Origin缺失fail-closed分支',
)
requirePattern(
  embedEntry,
  /startAssistantRuntimeSession\(\s*bootstrap\.publicId,\s*anonymousClientId,\s*parentOrigin,/s,
  '学生端向会话API传递父页面Origin',
)
requirePattern(
  heightBridge,
  /window\.parent\.postMessage\([\s\S]*targetOrigin,/,
  '高度消息精确targetOrigin',
)
forbidPattern(
  heightBridge,
  /postMessage\([\s\S]*,\s*['"]\*['"]\s*\)/,
  '高度消息postMessage通配符',
)

forbidPattern(
  `${runtimeAPI}\n${embedEntry}`,
  /\b(?:localStorage|sessionStorage)\s*\.\s*(?:getItem|setItem|removeItem|clear)\s*\(/,
  '浏览器持久化存储读写父Origin或运行令牌',
)

requirePattern(
  backendModel,
  /ParentOrigin\s+string\s+`json:"parent_origin"`/,
  'Go请求模型parent_origin字段',
)
requirePattern(
  backendSupport,
  /r\.Header\.Get\("Origin"\)/,
  '后端HTTP Origin校验',
)
requirePattern(
  backendSupport,
  /r\.Header\.Get\("Referer"\)/,
  '后端HTTP Referer校验',
)
requirePattern(
  backendSupport,
  /assistantRuntimeEmbedContextPrefix\s*\+\s*publicID/,
  '后端官方embed路径与public_id绑定',
)
requirePattern(
  backendSupport,
  /w\.Header\(\)\.Set\("Referrer-Policy",\s*"same-origin"\)/s,
  '后端same-origin响应头',
)
requirePattern(
  backendSupport,
  /<meta name="referrer" content="same-origin">/,
  '后端same-origin HTML meta',
)
requirePattern(
  backendHandler,
  /request\.ParentOrigin/,
  '处理器向服务传递真实父页面Origin',
)
forbidPattern(
  frameAncestorsFunction,
  /['"]'self'['"]/,
  'frame-ancestors隐式self',
)
requirePattern(
  backendSupport,
  /ErrAssistantDeploymentStoredPolicyInvalid[\s\S]*StatusServiceUnavailable/,
  '已保存策略损坏503映射',
)

// ==================== 功能开关 ====================

requirePattern(
  backendConfig,
  /GetBoolEnv\(\s*"COURSEWARE_ASSISTANT_ENABLED",\s*true,/s,
  '教师端教学智能体安全兼容默认值',
)
requirePattern(
  backendConfig,
  /GetBoolEnv\(\s*"COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED",\s*false,/s,
  '公开运行安全默认关闭',
)
requirePattern(
  backendConfig,
  /!cfg\.CoursewareAssistantEnabled[\s\S]*cfg\.CoursewareAssistantPublicRuntimeEnabled\s*=\s*false/,
  '教师端关闭时公开运行强制收敛',
)
requirePattern(
  backendRuntimeRoute,
  /sessionService\.SetPublicRuntimeEnabled\(\s*publicRuntimeEnabled,\s*\)/s,
  '公开开关注入运行会话服务',
)
requirePattern(
  backendSessionService,
  /case\s+models\.AssistantRuntimeSessionKindExternal:[\s\S]*!s\.publicRuntimeEnabled/s,
  'external会话类型总闸门',
)
requirePattern(
  backendSessionService,
  /case\s+models\.AssistantRuntimeSessionKindTeacherPreview:[\s\S]*return\s+nil/s,
  'teacher_preview独立运行通道',
)
requirePattern(
  backendAuthorization,
  /s\.validateSessionKindEnabled\(\s*session\.SessionKind,\s*\)/s,
  '已签发令牌数据库会话类型实时校验',
)

// ==================== 401自动恢复 ====================

requirePattern(
  embedEntry,
  /AssistantRuntimeAPIError/,
  '公开API结构化错误类型',
)
requirePattern(
  synchronizeFunction,
  /cause\s+instanceof\s+AssistantRuntimeAPIError/,
  '会话同步结构化错误判断',
)
requirePattern(
  synchronizeFunction,
  /cause\.status\s*===\s*401/,
  '会话同步HTTP 401识别',
)
requirePattern(
  synchronizeFunction,
  /restartSessionRef\.current\(\)/,
  '401触发新会话恢复',
)
requirePattern(
  embedEntry,
  /restartSessionRef\.current\s*=\s*\(\)\s*=>\s*\{[\s\S]*void\s+startSession\(true\)/,
  '401恢复调用带恢复标记的新会话',
)
requirePattern(
  startSessionFunction,
  /recoveringFromTokenExpiry\s*=\s*false/,
  '新建会话恢复模式参数',
)
requirePattern(
  startSessionFunction,
  /setMessages\(\[\]\)/,
  '恢复时清除旧会话乐观消息',
)
requirePattern(
  embedEntry,
  /上一条消息不会自动重发/,
  '恢复时不自动重发的学生提示',
)
forbidPattern(
  startSessionFunction,
  /streamAssistantRuntimeChat\(/,
  '新建会话流程自动重发聊天消息',
)

console.log(
  '开发单元25/26前端与后端公开运行永久协议检查：通过',
)
