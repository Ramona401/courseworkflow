/**
 * ai-assistants.ts — AI 助手 API 封装(TE-DNA 3.0 P0)
 *
 * 对应后端:
 *   - handlers/ai_assistant_handler.go
 *   - services/ai_assistant_service.go
 *   - models/ai_assistant.go
 *   - routes/routes_ai_assistant.go
 *
 * 三层架构:
 *   - 老师只看助手(本文件),不直接看组件库
 *   - 助手通过后端 AOCI 调用组件知识库
 *   - 组件库退居幕后
 *
 * 适用角色与权限:
 *   - admin           : 可创建/编辑/删除 system 助手 + 自己的 personal
 *   - senior_operator : 可创建/编辑/删除 本校 group 助手 + 自己的 personal
 *   - operator/viewer : 仅可创建/编辑/删除 自己的 personal
 *   - 所有人均可查看:system + 本校 group + 自己的 personal
 *
 * 响应格式说明:
 *   - client.ts 的响应拦截器已处理 code !== 0 的错误(抛出 Error)
 *   - 本文件所有 API 函数可直接返回 res.data.data,无需再判断 code
 *
 * v114 Batch 2 第 1 轮(2026-04-20):
 *   文件末尾追加 designChat SSE 函数 + 配套类型,对接后端
 *   POST /api/v1/ai-assistants/design/chat(对话式创作)
 *   仿 api/annotations.ts 的 aiFixAnnotation 样板编写
 */
import client from './client'

// ==================== 常量:来源与场景 ====================

/** 助手来源:system=系统预置 / group=教研员本校 / personal=个人私有 */
export type AssistantSource = 'system' | 'group' | 'personal'

/** 助手适用场景:对应后端 models.SceneXxx 常量 */
export type AssistantScene =
  | 'review_workbench'    // 独立全屏评审工作台
  | 'workshop_analyze'    // 备课工坊 — 教学分析阶段
  | 'workshop_design'     // 备课工坊 — 教学设计阶段
  | 'workshop_write'      // 备课工坊 — 教案撰写阶段
  | 'workshop_review'     // 备课工坊 — AI 评审阶段
  | 'workshop_revise'     // 备课工坊 — 修订定稿阶段

/** 场景中文展示名(前端 Selector/EditModal 用) */
export const ASSISTANT_SCENE_LABELS: Record<AssistantScene, string> = {
  review_workbench: '评审工作台',
  workshop_analyze: '工坊-教学分析',
  workshop_design: '工坊-教学设计',
  workshop_write: '工坊-教案撰写',
  workshop_review: '工坊-AI评审',
  workshop_revise: '工坊-修订定稿',
}

/** 来源中文展示名 */
export const ASSISTANT_SOURCE_LABELS: Record<AssistantSource, string> = {
  system: '系统',
  group: '本校',
  personal: '我的',
}

/** 来源对应的 emoji 前缀(与后端 5 条预置助手命名风格一致) */
export const ASSISTANT_SOURCE_EMOJI: Record<AssistantSource, string> = {
  system: '🏛️',
  group: '🏫',
  personal: '👤',
}

// ==================== 类型定义:列表项 ====================

/**
 * 助手列表项(对应后端 models.AIAssistantListItem)
 *
 * 与详情(AIAssistant)的差异:
 *   - scenes 已解析为字符串数组(详情里是 JSONB 字符串)
 *   - 附带展示辅助字段(creator_name/school_name/source_label)
 *   - 附带权限计算字段(can_edit/can_delete)
 *   - 附带当前场景匹配字段(is_default_here)
 */
export interface AIAssistantListItem {
  id: string
  name: string
  avatar_emoji: string
  description: string

  // 来源与展示
  source: AssistantSource
  source_label: string       // 后端映射的中文名:系统/本校/我的

  // 匹配维度
  subject: string            // 空字符串表示不限学科
  grade_range: string        // 空字符串表示不限年级
  scenes: AssistantScene[]   // 已解析的场景数组

  // 飞轮数据
  use_count: number
  avg_score: number | null

  // 状态
  is_active: boolean
  is_default_here: boolean   // 是否在当前查询场景下被标记为默认

  // 当前用户的权限
  can_edit: boolean
  can_delete: boolean

  // 归属展示
  creator_name: string       // 创建者姓名(system 助手可能为空)
  school_name: string        // 学校名称(group 助手时有值)

  // 时间
  created_at: string | null
  updated_at: string | null
}

/** 助手列表响应 */
export interface AIAssistantListResponse {
  assistants: AIAssistantListItem[]
  total: number
}

// ==================== 类型定义:详情 ====================

/**
 * 助手完整详情(对应后端 models.AIAssistant)
 *
 * 注意 scenes 是 JSONB 字符串,前端使用前需要 JSON.parse
 * 为简化前端调用,可以走 parseAssistantScenes 辅助函数
 */
export interface AIAssistant {
  id: string
  name: string
  avatar_emoji: string
  description: string

  // 来源与归属
  source: AssistantSource
  created_by: string | null
  organization_id: string | null
  group_id: string | null

  // 核心内容
  full_prompt: string
  knowledge_refs: string       // JSONB 字符串:引用的组件/教案 ID 数组

  // 匹配维度
  subject: string
  grade_range: string
  scenes: string               // JSONB 字符串,需要 JSON.parse 为 AssistantScene[]

  // 创作轨迹(P0.5 用)
  creation_conversation: string | null
  forked_from: string | null

  // 飞轮数据
  use_count: number
  avg_score: number | null

  // 状态与排序
  sort_order: number
  is_default_for_scene: string // JSONB 字符串:该助手默认匹配的场景列表
  is_active: boolean

  // 时间
  created_at: string | null
  updated_at: string | null
}

// ==================== 类型定义:创建/更新请求 ====================

/**
 * 创建助手请求(对应后端 models.CreateAIAssistantRequest)
 *
 * source 字段前端可以不传,后端会根据当前用户角色自动决定:
 *   - admin          → system
 *   - senior_operator → 前端指定 'group' / 'personal'(默认 personal)
 *   - operator/viewer → personal(不可改)
 *
 * 前端仅当 senior_operator 创建本校助手时才传 source='group'
 */
export interface CreateAIAssistantRequest {
  name: string
  avatar_emoji?: string
  description?: string
  source?: AssistantSource       // 可选,后端按角色校验/纠正
  full_prompt: string
  subject?: string
  grade_range?: string
  scenes: AssistantScene[]       // 至少 1 个场景
  forked_from?: string | null
}

/**
 * 更新助手请求(对应后端 models.UpdateAIAssistantRequest)
 *
 * 不允许改 source/归属,后端会忽略这些字段
 * is_active 可空,不传表示保持现状
 */
export interface UpdateAIAssistantRequest {
  name: string
  avatar_emoji?: string
  description?: string
  full_prompt: string
  subject?: string
  grade_range?: string
  scenes: AssistantScene[]
  is_active?: boolean
}

// ==================== 类型定义:列表查询参数 ====================

/** 列表查询参数 */
export interface ListAssistantsParams {
  /** 场景筛选(空=全部).Selector 组件会按当前场景过滤 */
  scene?: AssistantScene
  /** 学科筛选(空=全部) */
  subject?: string
  /** 年级筛选(空=全部) */
  grade?: string
  /** 是否只显示激活的 */
  only_active?: boolean
}

// ==================== API:列表 ====================

/**
 * 按场景和用户角色返回可见的助手列表
 *
 * 可见性规则(后端实现):
 *   - system 助手:所有人可见
 *   - group 助手:仅本校教师可见(通过 teaching_group_members 判定)
 *   - personal 助手:仅创建者可见
 *
 * 典型调用示例:
 *   评审工作台 Selector:listAssistants({scene: 'review_workbench'})
 *   工坊 design 阶段:listAssistants({scene: 'workshop_design', subject: '人工智能', grade: '7-9'})
 *   我的助手页:    listAssistants() — 不传参数,返回全部可见助手
 */
export async function listAssistants(params?: ListAssistantsParams): Promise<AIAssistantListResponse> {
  const query = new URLSearchParams()
  if (params?.scene) query.set('scene', params.scene)
  if (params?.subject) query.set('subject', params.subject)
  if (params?.grade) query.set('grade', params.grade)
  if (params?.only_active !== undefined) query.set('only_active', String(params.only_active))
  const qs = query.toString()
  const url = `/ai-assistants${qs ? '?' + qs : ''}`
  const res = await client.get<{ code: number; data: AIAssistantListResponse }>(url)
  return res.data.data!
}

// ==================== API:详情 ====================

/**
 * 获取助手详情(含 full_prompt 完整内容,用于编辑)
 *
 * 权限校验:后端会验证当前用户对该助手的可见性
 *   - 不可见时返回 403
 *   - 不存在时返回 404
 */
export async function getAssistant(id: string): Promise<AIAssistant> {
  const res = await client.get<{ code: number; data: AIAssistant }>(`/ai-assistants/${id}`)
  return res.data.data!
}

// ==================== API:创建 ====================

/**
 * 创建助手
 *
 * 后端会根据当前用户角色自动决定 source:
 *   - admin 不指定 source 时默认 'system'
 *   - senior_operator 默认 'personal',若想创建本校助手需显式传 source='group'
 *   - 其他角色强制为 'personal'
 *
 * 用户可见错误:
 *   - 400: 名称/提示词/场景缺失,或场景代码无效
 *   - 403: 角色不允许创建该 source 的助手
 */
export async function createAssistant(req: CreateAIAssistantRequest): Promise<AIAssistant> {
  const res = await client.post<{ code: number; data: AIAssistant }>('/ai-assistants', req)
  return res.data.data!
}

// ==================== API:更新 ====================

/**
 * 更新助手(全量更新,不是增量)
 *
 * 权限要求:
 *   - system 助手:仅 admin 可编辑
 *   - group 助手:仅本校 senior_operator 可编辑
 *   - personal 助手:仅创建者可编辑
 *
 * 返回 void,前端需要最新数据时应重新调用 getAssistant(id)
 */
export async function updateAssistant(id: string, req: UpdateAIAssistantRequest): Promise<void> {
  await client.put(`/ai-assistants/${id}`, req)
}

// ==================== API:删除 ====================

/**
 * 删除助手(硬删除)
 *
 * 特别注意:
 *   - system 助手禁止硬删除(后端返回 403,错误信息会提示"如需停用请修改 is_active")
 *   - 如需停用 system,走 updateAssistant 把 is_active=false
 */
export async function deleteAssistant(id: string): Promise<void> {
  await client.delete(`/ai-assistants/${id}`)
}

// ==================== API:Fork(复制到我的) ====================

/**
 * 将系统/本校助手复制一份到"我的"
 *
 * 复制后:
 *   - source = 'personal'(强制)
 *   - created_by = 当前用户
 *   - name 末尾自动追加"(我的副本)"
 *   - full_prompt / scenes 原样复制
 *   - forked_from 记录原助手 ID,便于追溯
 *
 * 典型场景:
 *   老师看到系统助手好用,点"复制到我的"后在自己的副本里做个性化调整
 */
export async function forkAssistant(sourceId: string): Promise<AIAssistant> {
  const res = await client.post<{ code: number; data: AIAssistant }>(`/ai-assistants/${sourceId}/fork`)
  return res.data.data!
}

// ==================== 辅助函数 ====================

/**
 * 解析助手详情中的 scenes 字段(JSONB 字符串 → 字符串数组)
 *
 * 详情接口返回的 scenes 是原始 JSONB 字符串(如 '["review_workbench"]'),
 * 列表接口返回的已经是数组,差异由后端处理,前端详情层需要解析
 */
export function parseAssistantScenes(scenesJSON: string): AssistantScene[] {
  if (!scenesJSON) return []
  try {
    const parsed = JSON.parse(scenesJSON)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((s): s is AssistantScene =>
      typeof s === 'string' && s in ASSISTANT_SCENE_LABELS,
    )
  } catch {
    return []
  }
}

/**
 * 解析助手 is_default_for_scene 字段(同 parseAssistantScenes)
 *
 * 用于在 EditModal 中展示"这个助手被设为哪些场景的默认"
 */
export function parseDefaultScenes(defaultScenesJSON: string): AssistantScene[] {
  return parseAssistantScenes(defaultScenesJSON)
}

/**
 * 按 sort_order + is_default_here 优先级排序列表
 *
 * 显示规则:
 *   1. is_default_here=true 的助手排最前(Selector 默认选中此项)
 *   2. 按 source 优先级排(system > group > personal)
 *   3. 同 source 内按 sort_order 升序
 *   4. 同 sort_order 按 use_count 降序(热度排序)
 */
export function sortAssistants(items: AIAssistantListItem[]): AIAssistantListItem[] {
  const sourceOrder: Record<AssistantSource, number> = {
    system: 1,
    group: 2,
    personal: 3,
  }
  return [...items].sort((a, b) => {
    // 1. is_default_here
    if (a.is_default_here !== b.is_default_here) {
      return a.is_default_here ? -1 : 1
    }
    // 2. source 优先级
    if (a.source !== b.source) {
      return sourceOrder[a.source] - sourceOrder[b.source]
    }
    // 3. use_count 降序(列表没有 sort_order,用热度代替)
    return b.use_count - a.use_count
  })
}

/**
 * 按来源分组列表(Selector 下拉菜单用)
 *
 * 返回值结构:
 *   {
 *     system:   [...],  // 🏛️ 系统
 *     group:    [...],  // 🏫 本校
 *     personal: [...],  // 👤 我的
 *   }
 */
export function groupAssistantsBySource(items: AIAssistantListItem[]): Record<AssistantSource, AIAssistantListItem[]> {
  const result: Record<AssistantSource, AIAssistantListItem[]> = {
    system: [],
    group: [],
    personal: [],
  }
  for (const item of items) {
    result[item.source].push(item)
  }
  return result
}

/* ================================================================
 *  v114 Batch 2:对话式助手创作 SSE 客户端
 * ================================================================
 *
 * 对应后端:
 *   POST /api/v1/ai-assistants/design/chat
 *   handlers/assistant_designer_handler.go → DesignChatRequest
 *
 * SSE 事件协议(与 handler 端注释完全一致):
 *   event: connected     → {"phase":"start"}
 *   event: searching     → {"reason":"..."}
 *   event: components    → {"components":[{id,name,library_type,...}]}
 *   event: chunk         → {"chunk":"..."}            (多次)
 *   event: draft_update  → {"draft":"..."}            (一次,done 之前)
 *   event: done          → {"reply":"...","draft":"...","referenced":[id,...]}
 *   event: error         → {"error":"..."}
 *
 * 设计要点:
 *   - 走原生 fetch + ReadableStream(axios 不支持 SSE)
 *   - 仿 annotations.ts 的 aiFixAnnotation 结构,事件驱动回调
 *   - 返回 {close} 句柄,允许组件卸载时中止连接(避免 React 警告)
 */

// -------------------- 请求体类型 --------------------

/**
 * 对话历史单条消息(与后端 services.DesignerMessage 对齐)
 * 后端只用 role + content 两个字段,这里保持一致
 */
export interface DesignerMessage {
  role: 'user' | 'assistant'
  content: string
}

/**
 * designChat 请求体(对应 handlers.DesignChatRequest)
 *
 * 关键字段说明:
 *   message       - 老师本轮输入
 *   history       - 历史对话(不含本轮),首轮传空数组
 *   subject       - 从 Modal 透传(后端构建组件库检索条件用)
 *   grade         - 从 Modal 透传
 *   scenes        - Modal 勾选的适用场景,可为空数组
 *   current_draft - 当前草稿(老师手动改过则传手动改后的,否则传上一轮 draft)
 */
export interface DesignChatRequest {
  message: string
  history: DesignerMessage[]
  subject?: string
  grade?: string
  scenes?: AssistantScene[]
  current_draft?: string
}

// -------------------- 回调事件载荷类型 --------------------

/**
 * components 事件里单个组件的轻量信息
 * 对应后端 services.ComponentBrief
 */
export interface DesignerComponentBrief {
  id: string
  name: string
  library_type: string      // 后端冗余了父 group 的 library_type
  library_name?: string     // 同上,中文名(可能为空)
  subject?: string
  grade_range?: string
  quality_score?: number
}

/**
 * SSE 事件回调集合
 *
 * 所有回调都是可选的,组件按需订阅:
 *   - 典型对话区只需 onChunk + onDone + onError
 *   - 完整体验还需 onConnected(显示"思考中")+ onSearching(显示"查库中")+ onComponents(展示参考)+ onDraftUpdate(同步右侧草稿)
 */
export interface DesignChatHandlers {
  /** 流建立(Nginx/后端已就绪),可用来切换对话区 loading 态 */
  onConnected?: () => void
  /** AI 决定查库时触发,reason 是后端给的一句话解释 */
  onSearching?: (reason: string) => void
  /** 组件库查询结果,briefs 通常 3-8 个 */
  onComponents?: (briefs: DesignerComponentBrief[]) => void
  /** AI 回复的流式片段(会被多次调用) */
  onChunk?: (chunk: string) => void
  /** 草稿更新(done 之前触发一次,携带完整最新草稿) */
  onDraftUpdate?: (draft: string) => void
  /** 流正常结束 — reply=AI 完整回复文字;draft=最终草稿;referenced=引用到的组件 ID 数组 */
  onDone?: (reply: string, draft: string, referenced: string[]) => void
  /** 任何错误(连接失败/后端 error 事件/JSON 解析异常) */
  onError?: (error: string) => void
}

/** designChat 返回的连接句柄(与 AIFixSSEConnection 风格一致) */
export interface DesignChatSSEConnection {
  /** 主动关闭连接(组件卸载或用户取消时调用) */
  close: () => void
}

// -------------------- 核心函数 --------------------

/**
 * designChat — 对话式助手创作 SSE 客户端
 *
 * 示例用法(在 DesignerPanel 内):
 *
 *   const conn = designChat(
 *     { message: '做一个严苛的初中 AI 审核员', history: [], subject: '人工智能', grade: '7-9' },
 *     {
 *       onConnected: () => setLoading(true),
 *       onSearching: (r) => setStatusTip(`🔍 ${r}`),
 *       onComponents: (cs) => setRefList(cs),
 *       onChunk: (c) => setReply(prev => prev + c),
 *       onDraftUpdate: (d) => setDraft(d),
 *       onDone: (reply, draft, ref) => { setLoading(false); /* 持久化等 *\/ },
 *       onError: (e) => { setLoading(false); setErrorMsg(e) },
 *     }
 *   )
 *   // 组件卸载时:conn.close()
 *
 * @param req      请求体
 * @param handlers 事件回调
 * @returns 连接句柄,含 close() 方法
 */
export function designChat(
  req: DesignChatRequest,
  handlers: DesignChatHandlers,
): DesignChatSSEConnection {
  let isClosed = false
  const controller = new AbortController()

  const token = localStorage.getItem('token') || ''

  // 注意:请求 URL 用绝对路径 /api/v1/...,与 aiFixAnnotation 保持一致
  // (axios baseURL 是 /api/v1,但 fetch 不走 axios,要手写完整路径)
  fetch('/api/v1/ai-assistants/design/chat', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify({
      message: req.message,
      history: req.history || [],
      subject: req.subject || '',
      grade: req.grade || '',
      scenes: req.scenes || [],
      current_draft: req.current_draft || '',
    }),
    signal: controller.signal,
  }).then(async (response) => {
    // 前置错误:后端走普通 JSON 而不是 SSE(鉴权失败/参数错误等)
    if (!response.ok) {
      let errMsg = `请求失败: HTTP ${response.status}`
      try {
        const data = await response.json()
        if (data?.message) errMsg = data.message
      } catch {
        // 解析失败保留默认错误
      }
      handlers.onError?.(errMsg)
      return
    }
    if (!response.body) {
      handlers.onError?.('浏览器不支持流式响应')
      return
    }

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (!isClosed) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      // SSE 事件以 \n\n 分隔
      const parts = buffer.split('\n\n')
      buffer = parts.pop() ?? ''

      for (const part of parts) {
        const lines = part.trim().split('\n')
        let eventType = ''
        let dataStr = ''

        for (const line of lines) {
          if (line.startsWith('event: ')) {
            eventType = line.slice(7).trim()
          } else if (line.startsWith('data: ')) {
            dataStr = line.slice(6).trim()
          }
        }

        if (!eventType || !dataStr) continue

        try {
          const data = JSON.parse(dataStr)
          switch (eventType) {
            case 'connected':
              handlers.onConnected?.()
              break
            case 'searching':
              handlers.onSearching?.(typeof data.reason === 'string' ? data.reason : '')
              break
            case 'components':
              if (Array.isArray(data.components)) {
                handlers.onComponents?.(data.components as DesignerComponentBrief[])
              }
              break
            case 'chunk':
              if (typeof data.chunk === 'string') handlers.onChunk?.(data.chunk)
              break
            case 'draft_update':
              if (typeof data.draft === 'string') handlers.onDraftUpdate?.(data.draft)
              break
            case 'done':
              handlers.onDone?.(
                typeof data.reply === 'string' ? data.reply : '',
                typeof data.draft === 'string' ? data.draft : '',
                Array.isArray(data.referenced) ? data.referenced as string[] : [],
              )
              isClosed = true
              break
            case 'error':
              handlers.onError?.(typeof data.error === 'string' ? data.error : 'AI 调用失败')
              isClosed = true
              break
          }
        } catch {
          // JSON 解析失败的事件忽略,不中断流
        }
      }
    }

    reader.cancel()
  }).catch((err) => {
    if (isClosed) return // 主动关闭不算错误
    handlers.onError?.(`连接失败: ${err?.message ?? '未知错误'}`)
  })

  return {
    close: () => {
      isClosed = true
      controller.abort()
    },
  }
}
