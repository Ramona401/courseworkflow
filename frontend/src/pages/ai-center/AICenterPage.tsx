/**
 * AICenterPage — 统一 AI API 管理中心（admin专属）v2.0
 *
 * v2.0 核心变更：
 *   1. 查询可用模型后，所有模型全部展示（多选卡片，可设为默认）
 *   2. 每个场景的"模型"字段改为下拉菜单，选项来自查询到的可用模型列表
 *   3. 两区域联动：查询模型后自动刷新场景Tab的下拉选项
 *
 * 路由：/ai-center（独立路由，admin专属）
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  getGlobalConfig, updateGlobalConfig,
  getSceneConfigs, updateSceneConfig,
  testConnection, listModels,
} from '@/api/ai-config'
import type { GlobalConfig, SceneConfig, UpdateSceneConfigRequest } from '@/api/ai-config'

// ==================== 供应商目录 ====================

interface AIProvider {
  id: string
  name: string
  logo: string
  baseURL: string
  docsURL: string
  models: string[]       // 该供应商的参考模型（静态参考）
  description: string
  isChinese: boolean
}

const AI_PROVIDERS: AIProvider[] = [
  {
    id: 'anthropic', name: 'Anthropic Claude', logo: '🤖',
    baseURL: 'https://api.anthropic.com/v1',
    docsURL: 'https://docs.anthropic.com',
    models: ['claude-opus-4-5', 'claude-sonnet-4-5', 'claude-haiku-4-5', 'claude-3-5-sonnet-20241022', 'claude-3-5-haiku-20241022'],
    description: '最强推理能力，长上下文，适合复杂任务', isChinese: false,
  },
  {
    id: 'openai', name: 'OpenAI', logo: '✨',
    baseURL: 'https://api.openai.com/v1',
    docsURL: 'https://platform.openai.com/docs',
    models: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-3.5-turbo', 'o1-preview', 'o1-mini'],
    description: 'GPT系列，综合能力强，生态最完善', isChinese: false,
  },
  {
    id: 'gemini', name: 'Google Gemini', logo: '💎',
    baseURL: 'https://generativelanguage.googleapis.com/v1beta/openai',
    docsURL: 'https://ai.google.dev/docs',
    models: ['gemini-2.0-flash', 'gemini-2.0-flash-lite', 'gemini-1.5-pro', 'gemini-1.5-flash'],
    description: '多模态能力突出，超长上下文支持', isChinese: false,
  },
  {
    id: 'deepseek', name: 'DeepSeek', logo: '🔍',
    baseURL: 'https://api.deepseek.com/v1',
    docsURL: 'https://platform.deepseek.com/docs',
    models: ['deepseek-chat', 'deepseek-reasoner'],
    description: '国产顶尖，代码和推理能力极强，性价比高', isChinese: true,
  },
  {
    id: 'qwen', name: '阿里云通义千问', logo: '🌟',
    baseURL: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    docsURL: 'https://help.aliyun.com/zh/dashscope',
    models: ['qwen-max', 'qwen-max-latest', 'qwen-plus', 'qwen-turbo', 'qwen-long'],
    description: '阿里云出品，中文理解强，企业服务稳定', isChinese: true,
  },
  {
    id: 'doubao', name: '字节豆包', logo: '🫘',
    baseURL: 'https://ark.cn-beijing.volces.com/api/v3',
    docsURL: 'https://www.volcengine.com/docs/82379',
    models: ['doubao-pro-256k', 'doubao-pro-128k', 'doubao-pro-32k', 'doubao-lite-128k', 'doubao-lite-32k'],
    description: '字节跳动出品，中文对话流畅，价格亲民', isChinese: true,
  },
  {
    id: 'moonshot', name: '月之暗面 Moonshot', logo: '🌙',
    baseURL: 'https://api.moonshot.cn/v1',
    docsURL: 'https://platform.moonshot.cn/docs',
    models: ['moonshot-v1-128k', 'moonshot-v1-32k', 'moonshot-v1-8k'],
    description: '超长上下文专家，128k窗口，适合长文档处理', isChinese: true,
  },
  {
    id: 'zhipu', name: '智谱 GLM', logo: '🧠',
    baseURL: 'https://open.bigmodel.cn/api/paas/v4',
    docsURL: 'https://open.bigmodel.cn/dev/howuse/introduction',
    models: ['glm-4', 'glm-4-flash', 'glm-4-plus', 'glm-4-long', 'glm-4-air'],
    description: '清华系，学术背景深厚，中文理解优秀', isChinese: true,
  },
  {
    id: 'ernie', name: '百度文心', logo: '🎯',
    baseURL: 'https://qianfan.baidubce.com/v2',
    docsURL: 'https://cloud.baidu.com/doc/WENXINWORKSHOP',
    models: ['ernie-4.5-turbo-128k', 'ernie-4.5-turbo-32k', 'ernie-3.5-128k', 'ernie-speed-128k'],
    description: '百度出品，知识图谱丰富，中文搜索增强', isChinese: true,
  },
  {
    id: 'spark', name: '讯飞星火', logo: '⚡',
    baseURL: 'https://spark-api-open.xf-yun.com/v1',
    docsURL: 'https://www.xfyun.cn/doc/spark/Web.html',
    models: ['generalv3.5', 'generalv3', 'generalv2', 'general'],
    description: '科大讯飞出品，语音+语言一体化，教育场景优化', isChinese: true,
  },
  {
    id: 'oneapi', name: '中转/聚合API', logo: '🔀',
    baseURL: 'https://your-oneapi-domain.com/v1',
    docsURL: 'https://github.com/songquanpeng/one-api',
    models: ['anthropic/claude-sonnet-4-5', 'openai/gpt-4o', 'deepseek/deepseek-chat'],
    description: '自建中转站（One API/New API），统一管理多个供应商', isChinese: false,
  },
  {
    id: 'ollama', name: 'Ollama 本地模型', logo: '🦙',
    baseURL: 'http://localhost:11434/v1',
    docsURL: 'https://ollama.com/docs',
    models: ['llama3.2', 'llama3.1', 'qwen2.5', 'deepseek-r1', 'mistral', 'phi3'],
    description: '本地离线运行，数据不出境，适合内网部署', isChinese: false,
  },
]

// ==================== 样式常量 ====================

const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  primaryBorder: 'rgba(79,123,232,0.25)',
  success: '#10B981',
  successLight: 'rgba(16,185,129,0.08)',
  danger: '#EF4444',
  dangerLight: 'rgba(239,68,68,0.08)',
  warning: '#F59E0B',
  warningLight: 'rgba(245,158,11,0.08)',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  bg: '#F9FAFB',
  white: '#FFFFFF',
  card: '#FFFFFF',
}

// ==================== Toast ====================

function Toast({ message, type, onClose }: {
  message: string; type: 'success' | 'error'; onClose: () => void
}) {
  useEffect(() => {
    const t = setTimeout(onClose, 3500)
    return () => clearTimeout(t)
  }, [onClose])
  return (
    <div style={{
      position: 'fixed', top: '24px', right: '24px', zIndex: 9999,
      padding: '12px 20px', borderRadius: '12px', color: '#fff',
      fontSize: '14px', fontWeight: 500,
      background: type === 'success'
        ? 'linear-gradient(135deg,#10B981,#059669)'
        : 'linear-gradient(135deg,#EF4444,#DC2626)',
      boxShadow: '0 4px 16px rgba(0,0,0,0.15)',
    }}>
      {type === 'success' ? '✓ ' : '✕ '}{message}
    </div>
  )
}

// ==================== 供应商列表项 ====================

function ProviderItem({ provider, selected, onSelect }: {
  provider: AIProvider; selected: boolean; onSelect: (p: AIProvider) => void
}) {
  const [hovered, setHovered] = useState(false)
  return (
    <button
      onClick={() => onSelect(provider)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        width: '100%', display: 'flex', alignItems: 'center', gap: '10px',
        padding: '9px 12px', borderRadius: '10px', border: 'none',
        cursor: 'pointer', textAlign: 'left',
        background: selected ? C.primaryLight : hovered ? C.bg : 'transparent',
        transition: 'all 150ms ease',
      }}
    >
      <span style={{ fontSize: '18px', width: '24px', textAlign: 'center', flexShrink: 0 }}>
        {provider.logo}
      </span>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{
          fontSize: '13px', fontWeight: selected ? 600 : 400,
          color: selected ? C.primary : C.text,
          overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        }}>{provider.name}</div>
      </div>
      {selected && (
        <div style={{ width: '6px', height: '6px', borderRadius: '50%', background: C.primary, flexShrink: 0 }} />
      )}
    </button>
  )
}

// ==================== 模型下拉选择器 ====================
// 用于场景配置中选择模型，支持：继承全局 / 可用模型列表 / 手动输入

function ModelSelect({
  value,
  onChange,
  availableModels,
  placeholder,
}: {
  value: string | null
  onChange: (v: string | null) => void
  availableModels: string[]   // 从API查询到的可用模型ID列表
  placeholder?: string
}) {
  // 判断当前值是否在可用列表中
  const isCustom = value !== null && value !== '' && !availableModels.includes(value)
  const [showCustomInput, setShowCustomInput] = useState(isCustom)
  const [customVal, setCustomVal] = useState(isCustom ? value : '')

  const handleSelectChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const v = e.target.value
    if (v === '__custom__') {
      // 切换到手动输入
      setShowCustomInput(true)
      setCustomVal('')
      onChange(null)
    } else if (v === '__inherit__') {
      // 继承全局
      setShowCustomInput(false)
      setCustomVal('')
      onChange(null)
    } else {
      // 选择某个模型
      setShowCustomInput(false)
      setCustomVal('')
      onChange(v)
    }
  }

  // 当前下拉的值
  const selectValue = showCustomInput
    ? '__custom__'
    : (value === null || value === '')
      ? '__inherit__'
      : availableModels.includes(value) ? value : '__custom__'

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
      <select
        value={selectValue}
        onChange={handleSelectChange}
        style={{
          width: '100%', padding: '8px 12px', borderRadius: '8px',
          border: `1px solid ${C.border}`, fontSize: '13px',
          outline: 'none', background: C.white, cursor: 'pointer',
          color: C.text,
        }}
        onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
        onBlur={e => { e.currentTarget.style.borderColor = C.border }}
      >
        {/* 继承全局选项 */}
        <option value="__inherit__">— 继承全局配置 —</option>

        {/* 可用模型分组 */}
        {availableModels.length > 0 && (
          <optgroup label={`可用模型（${availableModels.length}个）`}>
            {availableModels.map(m => (
              <option key={m} value={m}>{m}</option>
            ))}
          </optgroup>
        )}

        {/* 手动输入选项 */}
        <option value="__custom__">✏️ 手动输入模型名...</option>
      </select>

      {/* 手动输入框（当选择"手动输入"时显示）*/}
      {showCustomInput && (
        <input
          value={customVal}
          onChange={e => {
            setCustomVal(e.target.value)
            onChange(e.target.value || null)
          }}
          placeholder={placeholder || '请输入模型名称'}
          autoFocus
          style={{
            width: '100%', padding: '7px 12px', borderRadius: '8px',
            border: `1.5px solid ${C.primary}`, fontSize: '13px',
            outline: 'none', boxSizing: 'border-box', fontFamily: 'monospace',
          }}
        />
      )}
    </div>
  )
}

// ==================== 主组件 ====================

export default function AICenterPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const fromPath: string = (location.state as { from?: string })?.from || '/'

  // 供应商
  const [selectedProvider, setSelectedProvider] = useState<AIProvider>(AI_PROVIDERS[0])

  // 全局配置
  const [globalConfig, setGlobalConfig] = useState<GlobalConfig | null>(null)
  const [form, setForm] = useState({
    api_base_url: '',
    api_key: '',
    default_model: '',
    temperature: '0.3',
    max_tokens: '64000',
  })
  const [showKey, setShowKey] = useState(false)
  const [saving, setSaving] = useState(false)

  // 连通性测试
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<{
    success: boolean; message: string; latency_ms: number; model: string
  } | null>(null)

  // =====================================================
  // 可用模型列表 — 核心状态
  // availableModels: 从API查询到的所有可用模型ID（字符串数组）
  // 这个列表同时用于：
  //   1. 连接配置Tab展示所有可用模型
  //   2. 场景配置Tab每个场景的模型下拉选项
  // =====================================================
  const [modelsLoading, setModelsLoading] = useState(false)
  const [availableModels, setAvailableModels] = useState<string[]>([])
  const [modelsError, setModelsError] = useState<string | null>(null)
  const [modelsQueried, setModelsQueried] = useState(false) // 是否已查询过

  // 场景配置
  const [scenes, setScenes] = useState<SceneConfig[]>([])
  const [editingScene, setEditingScene] = useState<string | null>(null)
  const [sceneForm, setSceneForm] = useState<UpdateSceneConfigRequest>({})
  const [sceneSaving, setSceneSaving] = useState(false)

  // 通用
  const [loading, setLoading] = useState(true)
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)
  const showToast = (m: string, t: 'success' | 'error') => setToast({ message: m, type: t })

  // Tab
  const [activeTab, setActiveTab] = useState<'connection' | 'scenes'>('connection')

  // ==================== 加载数据 ====================

  const loadData = useCallback(async () => {
    try {
      setLoading(true)
      const [gc, sc] = await Promise.all([getGlobalConfig(), getSceneConfigs()])
      setGlobalConfig(gc)
      setForm({
        api_base_url: gc.api_base_url || '',
        api_key: '',
        default_model: gc.default_model || '',
        temperature: gc.temperature || '0.3',
        max_tokens: gc.max_tokens || '64000',
      })
      setScenes(sc || [])
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '加载失败', 'error')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { loadData() }, [loadData])

  // ==================== 切换供应商 ====================

  const handleSelectProvider = (p: AIProvider) => {
    setSelectedProvider(p)
    setForm(prev => ({ ...prev, api_base_url: p.baseURL }))
    setTestResult(null)
    // 切换供应商时清空已查询的模型（因为Key和地址都可能变了）
    setAvailableModels([])
    setModelsQueried(false)
    setModelsError(null)
  }

  // ==================== 保存全局配置 ====================

  const handleSave = async () => {
    try {
      setSaving(true)
      const result = await updateGlobalConfig(form)
      setGlobalConfig(result)
      setForm(prev => ({ ...prev, api_key: '' }))
      showToast('配置保存成功', 'success')
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '保存失败', 'error')
    } finally {
      setSaving(false)
    }
  }

  // ==================== 测试连接 ====================

  const handleTest = async () => {
    try {
      setTesting(true)
      setTestResult(null)
      const r = await testConnection()
      setTestResult({ success: r.success, message: r.message, latency_ms: r.latency_ms, model: r.model })
      showToast(r.success ? '连接测试成功！' : '连接测试失败', r.success ? 'success' : 'error')
    } catch (err: unknown) {
      setTestResult({ success: false, message: err instanceof Error ? err.message : '请求失败', latency_ms: 0, model: '' })
    } finally {
      setTesting(false)
    }
  }

  // ==================== 查询可用模型 ====================
  // 查询完成后 availableModels 自动填充到场景配置下拉菜单

  const handleListModels = async () => {
    try {
      setModelsLoading(true)
      setModelsError(null)
      const r = await listModels()
      const ids = (r.models || []).map(m => m.id)
      setAvailableModels(ids)
      setModelsQueried(true)
      showToast(`查询成功，共 ${ids.length} 个可用模型，已同步到场景配置下拉菜单`, 'success')
      // 查询完模型后自动切换到场景配置Tab，方便用户直接配置
      // 不自动切换，让用户先看清楚模型列表
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '查询失败'
      setModelsError(msg)
      setAvailableModels([])
      setModelsQueried(true)
    } finally {
      setModelsLoading(false)
    }
  }

  // ==================== 场景配置 ====================

  const handleEditScene = (scene: SceneConfig) => {
    setEditingScene(scene.scene_code)
    setSceneForm({
      model: scene.model,
      temperature: scene.temperature,
      max_tokens: scene.max_tokens,
      is_active: scene.is_active,
    })
  }

  const handleSaveScene = async (code: string) => {
    try {
      setSceneSaving(true)
      const result = await updateSceneConfig(code, sceneForm)
      setScenes(result || [])
      setEditingScene(null)
      showToast(`场景 "${code}" 已保存`, 'success')
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '保存失败', 'error')
    } finally {
      setSceneSaving(false)
    }
  }

  // ==================== 加载中 ====================

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', background: C.bg, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div style={{ textAlign: 'center' }}>
          <div style={{
            width: '36px', height: '36px', margin: '0 auto 16px',
            border: `3px solid ${C.primary}`, borderTopColor: 'transparent',
            borderRadius: '50%', animation: 'spin 0.8s linear infinite',
          }} />
          <style>{`@keyframes spin{to{transform:rotate(360deg)}}`}</style>
          <div style={{ color: C.textMuted, fontSize: '14px' }}>加载 AI 配置...</div>
        </div>
      </div>
    )
  }

  // ==================== 渲染 ====================

  return (
    <div style={{ minHeight: '100vh', background: 'linear-gradient(135deg,#EEF2FF 0%,#FAFBFC 60%,#F0FDF4 100%)' }}>
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      {/* 顶部导航 */}
      <header style={{
        height: '64px', position: 'sticky', top: 0, zIndex: 100,
        background: 'rgba(255,255,255,0.88)', backdropFilter: 'blur(20px)',
        borderBottom: `1px solid ${C.border}`,
        display: 'flex', alignItems: 'center', padding: '0 32px', gap: '16px',
      }}>
        <button
          onClick={() => navigate(fromPath)}
          style={{
            display: 'flex', alignItems: 'center', gap: '6px',
            padding: '8px 16px', borderRadius: '8px',
            border: `1px solid ${C.border}`, background: C.white,
            fontSize: '14px', color: C.textSec, cursor: 'pointer',
          }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.white }}
        >
          {'<- 返回'}
        </button>

        <div style={{ flex: 1, textAlign: 'center' }}>
          <h1 style={{ fontSize: '18px', fontWeight: 700, color: C.text, margin: 0 }}>
            🤖 AI API 管理中心
          </h1>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>
            统一管理大模型供应商与场景配置
          </div>
        </div>

        <div style={{
          padding: '6px 14px', borderRadius: '20px', fontSize: '12px', fontWeight: 600,
          background: globalConfig?.api_key_set ? 'rgba(16,185,129,0.1)' : 'rgba(239,68,68,0.1)',
          color: globalConfig?.api_key_set ? C.success : C.danger,
        }}>
          {globalConfig?.api_key_set ? '✓ API Key 已配置' : '⚠ API Key 未配置'}
        </div>
      </header>

      {/* 主体：左侧供应商 + 右侧配置 */}
      <div style={{
        display: 'flex', maxWidth: '1280px', margin: '0 auto',
        padding: '28px 24px', gap: '24px', alignItems: 'flex-start',
      }}>

        {/* 左侧供应商选择器 */}
        <aside style={{
          width: '260px', flexShrink: 0,
          background: C.card, borderRadius: '16px',
          border: `1px solid ${C.border}`,
          boxShadow: '0 2px 8px rgba(0,0,0,0.04)',
          overflow: 'hidden',
          position: 'sticky', top: '88px',
        }}>
          <div style={{
            padding: '14px 20px', borderBottom: `1px solid ${C.border}`,
            fontSize: '13px', fontWeight: 600, color: C.textSec,
          }}>
            选择供应商
          </div>
          <div style={{ padding: '8px 10px' }}>
            <div style={{ fontSize: '11px', color: C.textMuted, padding: '4px 8px', fontWeight: 600 }}>
              🌍 国际
            </div>
            {AI_PROVIDERS.filter(p => !p.isChinese).map(p => (
              <ProviderItem key={p.id} provider={p} selected={selectedProvider.id === p.id} onSelect={handleSelectProvider} />
            ))}
          </div>
          <div style={{ height: '1px', background: C.border, margin: '4px 16px' }} />
          <div style={{ padding: '8px 10px 12px' }}>
            <div style={{ fontSize: '11px', color: C.textMuted, padding: '4px 8px', fontWeight: 600 }}>
              🇨🇳 国内
            </div>
            {AI_PROVIDERS.filter(p => p.isChinese).map(p => (
              <ProviderItem key={p.id} provider={p} selected={selectedProvider.id === p.id} onSelect={handleSelectProvider} />
            ))}
          </div>
        </aside>

        {/* 右侧配置区 */}
        <div style={{ flex: 1, minWidth: 0 }}>

          {/* 供应商横幅 */}
          <div style={{
            background: C.card, borderRadius: '16px',
            border: `1px solid ${C.border}`,
            padding: '18px 24px', marginBottom: '20px',
            boxShadow: '0 2px 8px rgba(0,0,0,0.04)',
            display: 'flex', alignItems: 'center', gap: '14px',
          }}>
            <div style={{
              width: '48px', height: '48px', borderRadius: '12px',
              background: 'linear-gradient(135deg,#EEF2FF,#E0E7FF)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: '24px', flexShrink: 0,
            }}>{selectedProvider.logo}</div>
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: '17px', fontWeight: 700, color: C.text }}>
                {selectedProvider.name}
              </div>
              <div style={{ fontSize: '13px', color: C.textSec, marginTop: '2px' }}>
                {selectedProvider.description}
              </div>
            </div>
            <button
              onClick={() => window.open(selectedProvider.docsURL, '_blank', 'noopener,noreferrer')}
              style={{
                padding: '7px 14px', borderRadius: '8px',
                border: `1px solid ${C.border}`, background: C.bg,
                fontSize: '13px', color: C.primary, cursor: 'pointer', fontWeight: 500,
              }}
              onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.primaryLight }}
              onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
            >
              查看文档
            </button>
          </div>

          {/* Tab切换 */}
          <div style={{
            display: 'flex', gap: '4px', marginBottom: '16px',
            background: C.bg, borderRadius: '10px', padding: '4px',
            border: `1px solid ${C.border}`, width: 'fit-content',
          }}>
            {(['connection', 'scenes'] as const).map(tab => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                style={{
                  padding: '8px 20px', borderRadius: '8px', border: 'none', cursor: 'pointer',
                  fontSize: '14px', fontWeight: activeTab === tab ? 600 : 400,
                  color: activeTab === tab ? C.primary : C.textSec,
                  background: activeTab === tab ? C.white : 'transparent',
                  boxShadow: activeTab === tab ? '0 1px 4px rgba(0,0,0,0.08)' : 'none',
                  transition: 'all 150ms ease',
                }}
              >
                {tab === 'connection' ? '🔌 连接配置' : (
                  <span style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                    ⚙️ 场景配置
                    {/* 已查询到模型时，显示模型数量提示 */}
                    {modelsQueried && availableModels.length > 0 && (
                      <span style={{
                        padding: '1px 7px', borderRadius: '10px', fontSize: '11px',
                        background: C.successLight, color: C.success, fontWeight: 600,
                      }}>
                        {availableModels.length}个模型可选
                      </span>
                    )}
                  </span>
                )}
              </button>
            ))}
          </div>

          {/* ===== Tab: 连接配置 ===== */}
          {activeTab === 'connection' && (
            <div style={{
              background: C.card, borderRadius: '16px',
              border: `1px solid ${C.border}`,
              boxShadow: '0 2px 8px rgba(0,0,0,0.04)',
            }}>
              <div style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}` }}>
                <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>API 连接配置</div>
                <div style={{ fontSize: '13px', color: C.textSec, marginTop: '3px' }}>
                  配置 {selectedProvider.name} 的访问地址和密钥
                </div>
              </div>

              <div style={{ padding: '24px' }}>
                {/* API Base URL */}
                <div style={{ marginBottom: '16px' }}>
                  <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                    API Base URL
                  </label>
                  <input
                    value={form.api_base_url}
                    onChange={e => setForm(p => ({ ...p, api_base_url: e.target.value }))}
                    placeholder={selectedProvider.baseURL}
                    style={{
                      width: '100%', padding: '10px 14px', borderRadius: '10px',
                      border: `1px solid ${C.border}`, fontSize: '14px',
                      outline: 'none', boxSizing: 'border-box', background: C.white,
                      fontFamily: 'monospace',
                    }}
                    onFocus={e => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = `0 0 0 3px ${C.primaryLight}` }}
                    onBlur={e => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}
                  />
                  <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '3px' }}>
                    默认地址：{selectedProvider.baseURL}
                  </div>
                </div>

                {/* API Key */}
                <div style={{ marginBottom: '16px' }}>
                  <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                    API Key
                    {globalConfig?.api_key_set && (
                      <span style={{ fontWeight: 400, color: C.textMuted, marginLeft: '8px', fontSize: '12px' }}>
                        当前：{globalConfig.api_key}（留空不修改）
                      </span>
                    )}
                  </label>
                  <div style={{ position: 'relative' }}>
                    <input
                      type={showKey ? 'text' : 'password'}
                      value={form.api_key}
                      onChange={e => setForm(p => ({ ...p, api_key: e.target.value }))}
                      placeholder={globalConfig?.api_key_set ? '留空表示不修改' : `请输入 ${selectedProvider.name} API Key`}
                      style={{
                        width: '100%', padding: '10px 44px 10px 14px', borderRadius: '10px',
                        border: `1px solid ${C.border}`, fontSize: '14px',
                        outline: 'none', boxSizing: 'border-box', background: C.white, fontFamily: 'monospace',
                      }}
                      onFocus={e => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = `0 0 0 3px ${C.primaryLight}` }}
                      onBlur={e => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}
                    />
                    <button
                      onClick={() => setShowKey(p => !p)}
                      style={{ position: 'absolute', right: '12px', top: '50%', transform: 'translateY(-50%)', background: 'none', border: 'none', cursor: 'pointer', color: C.textMuted, fontSize: '16px' }}
                    >
                      {showKey ? '🙈' : '👁'}
                    </button>
                  </div>
                </div>

                {/* 温度 + Token（全局默认模型改为从可用模型中选）*/}
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 140px 160px', gap: '14px', marginBottom: '20px' }}>
                  <div>
                    <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                      默认模型
                      {availableModels.length > 0 && (
                        <span style={{ fontWeight: 400, color: C.success, marginLeft: '6px', fontSize: '11px' }}>
                          ({availableModels.length}个可用)
                        </span>
                      )}
                    </label>
                    {/* 全局默认模型也使用下拉选择器 */}
                    <ModelSelect
                      value={form.default_model || null}
                      onChange={v => setForm(p => ({ ...p, default_model: v || '' }))}
                      availableModels={availableModels}
                      placeholder="请输入或选择默认模型"
                    />
                  </div>
                  <div>
                    <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>温度</label>
                    <input
                      type="number" step="0.1" min="0" max="2"
                      value={form.temperature}
                      onChange={e => setForm(p => ({ ...p, temperature: e.target.value }))}
                      style={{ width: '100%', padding: '10px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: C.white }}
                      onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                      onBlur={e => { e.currentTarget.style.borderColor = C.border }}
                    />
                  </div>
                  <div>
                    <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>Max Tokens</label>
                    <input
                      type="number" step="1000"
                      value={form.max_tokens}
                      onChange={e => setForm(p => ({ ...p, max_tokens: e.target.value }))}
                      style={{ width: '100%', padding: '10px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: C.white }}
                      onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                      onBlur={e => { e.currentTarget.style.borderColor = C.border }}
                    />
                  </div>
                </div>

                {/* ===== 可用模型展示区（v2.0核心：所有模型全部展示）===== */}
                <div style={{
                  borderRadius: '14px',
                  border: `1px solid ${modelsQueried && !modelsError ? 'rgba(16,185,129,0.25)' : C.border}`,
                  background: modelsQueried && !modelsError ? 'rgba(16,185,129,0.03)' : C.bg,
                  marginBottom: '20px', overflow: 'hidden',
                }}>
                  {/* 标题行 */}
                  <div style={{
                    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                    padding: '14px 18px',
                    borderBottom: modelsQueried ? `1px solid ${modelsError ? 'rgba(239,68,68,0.15)' : 'rgba(16,185,129,0.15)'}` : 'none',
                  }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <span style={{ fontSize: '15px' }}>📋</span>
                      <span style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>
                        {modelsLoading
                          ? '查询中...'
                          : modelsError
                            ? '查询失败'
                            : modelsQueried
                              ? `当前 Key 可用模型（${availableModels.length} 个）`
                              : '可用模型列表'
                        }
                      </span>
                      {modelsQueried && !modelsError && (
                        <span style={{
                          padding: '2px 8px', borderRadius: '10px', fontSize: '11px',
                          background: C.successLight, color: C.success, fontWeight: 600,
                        }}>
                          已同步到场景配置下拉菜单
                        </span>
                      )}
                    </div>
                    <button
                      onClick={handleListModels}
                      disabled={modelsLoading}
                      style={{
                        padding: '7px 16px', borderRadius: '8px', border: 'none',
                        background: modelsLoading ? C.textMuted : 'linear-gradient(135deg,#10B981,#059669)',
                        color: '#fff', fontSize: '13px', fontWeight: 600,
                        cursor: modelsLoading ? 'not-allowed' : 'pointer',
                      }}
                    >
                      {modelsLoading ? '查询中...' : modelsQueried ? '重新查询' : '查询可用模型'}
                    </button>
                  </div>

                  {/* 错误提示 */}
                  {modelsError && (
                    <div style={{ padding: '14px 18px', fontSize: '13px', color: C.danger }}>
                      ⚠ {modelsError}
                    </div>
                  )}

                  {/* 未查询时的提示 */}
                  {!modelsQueried && !modelsLoading && (
                    <div style={{ padding: '20px 18px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>
                      点击"查询可用模型"获取当前 API Key 下所有可用模型
                      <br />
                      <span style={{ fontSize: '12px', marginTop: '4px', display: 'block' }}>
                        查询后模型列表将同步到场景配置的下拉菜单
                      </span>
                    </div>
                  )}

                  {/* 模型列表：全部展示为标签卡片，点击填入默认模型 */}
                  {!modelsError && availableModels.length > 0 && (
                    <div style={{ padding: '16px 18px' }}>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
                        {availableModels.map(m => {
                          const isDefault = form.default_model === m
                          return (
                            <button
                              key={m}
                              onClick={() => {
                                setForm(p => ({ ...p, default_model: m }))
                                showToast(`默认模型已设为：${m}`, 'success')
                              }}
                              title="点击设为默认模型"
                              style={{
                                padding: '6px 14px', borderRadius: '8px',
                                border: `1.5px solid ${isDefault ? C.primary : C.border}`,
                                background: isDefault ? C.primaryLight : C.white,
                                color: isDefault ? C.primary : C.text,
                                fontSize: '13px', fontFamily: 'monospace',
                                cursor: 'pointer', transition: 'all 150ms ease',
                                fontWeight: isDefault ? 700 : 400,
                                display: 'flex', alignItems: 'center', gap: '6px',
                              }}
                              onMouseEnter={e => {
                                if (!isDefault) {
                                  ;(e.currentTarget as HTMLElement).style.borderColor = C.primary
                                  ;(e.currentTarget as HTMLElement).style.background = C.primaryLight
                                  ;(e.currentTarget as HTMLElement).style.color = C.primary
                                }
                              }}
                              onMouseLeave={e => {
                                if (!isDefault) {
                                  ;(e.currentTarget as HTMLElement).style.borderColor = C.border
                                  ;(e.currentTarget as HTMLElement).style.background = C.white
                                  ;(e.currentTarget as HTMLElement).style.color = C.text
                                }
                              }}
                            >
                              {isDefault && <span style={{ fontSize: '10px' }}>✓</span>}
                              {m}
                            </button>
                          )
                        })}
                      </div>
                      <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '10px' }}>
                        点击模型可设为全局默认模型 · 共 {availableModels.length} 个 · 以上模型已同步到场景配置下拉菜单
                      </div>
                    </div>
                  )}
                </div>

                {/* 测试结果 */}
                {testResult && (
                  <div style={{
                    padding: '16px', borderRadius: '12px', marginBottom: '20px',
                    background: testResult.success ? C.successLight : C.dangerLight,
                    border: `1px solid ${testResult.success ? 'rgba(16,185,129,0.3)' : 'rgba(239,68,68,0.3)'}`,
                  }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '8px' }}>
                      <span>{testResult.success ? '✅' : '❌'}</span>
                      <span style={{ fontWeight: 600, color: testResult.success ? C.success : C.danger }}>
                        {testResult.success ? '连接成功' : '连接失败'}
                      </span>
                      {testResult.latency_ms > 0 && (
                        <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: 'rgba(0,0,0,0.06)', color: C.textSec }}>
                          {testResult.latency_ms}ms
                        </span>
                      )}
                    </div>
                    <div style={{ fontSize: '13px', color: C.text }}>{testResult.message}</div>
                    {testResult.model && (
                      <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>模型：{testResult.model}</div>
                    )}
                    <button
                      onClick={() => setTestResult(null)}
                      style={{ marginTop: '10px', padding: '4px 12px', borderRadius: '6px', border: `1px solid ${C.border}`, background: C.white, fontSize: '12px', color: C.textSec, cursor: 'pointer' }}
                    >
                      关闭
                    </button>
                  </div>
                )}

                {/* 操作按钮 */}
                <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end', flexWrap: 'wrap' }}>
                  <button
                    onClick={handleTest}
                    disabled={testing}
                    style={{
                      padding: '10px 20px', borderRadius: '10px', border: 'none',
                      background: testing ? C.textMuted : 'linear-gradient(135deg,#F59E0B,#D97706)',
                      color: '#fff', fontSize: '14px', fontWeight: 600,
                      cursor: testing ? 'not-allowed' : 'pointer',
                    }}
                  >
                    {testing ? '测试中...' : '🔌 测试连接'}
                  </button>
                  <button
                    onClick={handleSave}
                    disabled={saving}
                    style={{
                      padding: '10px 24px', borderRadius: '10px', border: 'none',
                      background: saving ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`,
                      color: '#fff', fontSize: '14px', fontWeight: 600,
                      cursor: saving ? 'not-allowed' : 'pointer',
                    }}
                  >
                    {saving ? '保存中...' : '💾 保存配置'}
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* ===== Tab: 场景配置 ===== */}
          {activeTab === 'scenes' && (
            <div style={{
              background: C.card, borderRadius: '16px',
              border: `1px solid ${C.border}`,
              boxShadow: '0 2px 8px rgba(0,0,0,0.04)',
            }}>
              <div style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}` }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <div>
                    <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>Pipeline 场景配置</div>
                    <div style={{ fontSize: '13px', color: C.textSec, marginTop: '3px' }}>
                      每个场景可独立指定模型，留空则继承全局配置
                    </div>
                  </div>
                  {/* 若未查询模型，显示提示 */}
                  {!modelsQueried && (
                    <div style={{
                      padding: '8px 14px', borderRadius: '10px',
                      background: C.warningLight,
                      border: `1px solid rgba(245,158,11,0.25)`,
                      fontSize: '12px', color: C.warning, fontWeight: 500,
                    }}>
                      💡 先在"连接配置"查询可用模型，下拉选项将自动填充
                    </div>
                  )}
                  {modelsQueried && availableModels.length > 0 && (
                    <div style={{
                      padding: '8px 14px', borderRadius: '10px',
                      background: C.successLight,
                      border: `1px solid rgba(16,185,129,0.25)`,
                      fontSize: '12px', color: C.success, fontWeight: 500,
                    }}>
                      ✓ 已加载 {availableModels.length} 个可用模型
                    </div>
                  )}
                </div>
              </div>

              <div style={{ padding: '16px 24px', display: 'flex', flexDirection: 'column', gap: '10px' }}>
                {scenes.map(scene => {
                  const isEditing = editingScene === scene.scene_code
                  return (
                    <div key={scene.scene_code} style={{
                      padding: '16px 20px', borderRadius: '12px',
                      border: `1px solid ${isEditing ? C.primary : C.border}`,
                      background: isEditing ? C.primaryLight : C.bg,
                      transition: 'all 200ms ease',
                    }}>
                      {/* 场景头部 */}
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                          <div style={{
                            width: '8px', height: '8px', borderRadius: '50%',
                            background: scene.is_active ? C.success : C.textMuted,
                          }} />
                          <span style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>
                            {scene.scene_name}
                          </span>
                          <span style={{
                            fontSize: '12px', color: C.textMuted, fontFamily: 'monospace',
                            padding: '2px 8px', background: 'rgba(0,0,0,0.05)', borderRadius: '4px',
                          }}>
                            {scene.scene_code}
                          </span>
                        </div>
                        {!isEditing && (
                          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                            {/* 当前模型显示 — 若在可用列表中高亮 */}
                            <span style={{
                              fontSize: '12px', fontFamily: 'monospace',
                              color: scene.model && availableModels.includes(scene.model) ? C.success : C.textMuted,
                            }}>
                              {scene.model || '继承全局'} · T:{scene.temperature ?? '继承'} · Max:{scene.max_tokens ?? '继承'}
                            </span>
                            <button
                              onClick={() => handleEditScene(scene)}
                              style={{
                                padding: '5px 14px', borderRadius: '8px',
                                border: `1px solid ${C.border}`, background: C.white,
                                fontSize: '12px', fontWeight: 500, color: C.primary, cursor: 'pointer',
                              }}
                            >
                              编辑
                            </button>
                          </div>
                        )}
                      </div>

                      {/* 编辑表单 */}
                      {isEditing && (
                        <div style={{ marginTop: '16px' }}>
                          <div style={{ display: 'grid', gridTemplateColumns: '1fr 120px 140px 100px', gap: '12px', marginBottom: '14px' }}>
                            {/* 模型下拉选择器（v2.0核心：使用可用模型列表作为选项）*/}
                            <div>
                              <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '4px' }}>
                                模型
                                {availableModels.length > 0 && (
                                  <span style={{ fontWeight: 400, color: C.success, marginLeft: '4px' }}>
                                    ({availableModels.length}个可选)
                                  </span>
                                )}
                              </label>
                              <ModelSelect
                                value={sceneForm.model ?? null}
                                onChange={v => setSceneForm(p => ({ ...p, model: v }))}
                                availableModels={availableModels}
                                placeholder="输入模型名称"
                              />
                            </div>
                            <div>
                              <label style={{ display: 'block', fontSize: '12px', color: C.textSec, marginBottom: '4px' }}>温度</label>
                              <input
                                type="number" step="0.1" min="0" max="2"
                                value={sceneForm.temperature ?? ''}
                                onChange={e => setSceneForm(p => ({ ...p, temperature: e.target.value === '' ? null : parseFloat(e.target.value) }))}
                                placeholder="继承全局"
                                style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', boxSizing: 'border-box' }}
                                onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                                onBlur={e => { e.currentTarget.style.borderColor = C.border }}
                              />
                            </div>
                            <div>
                              <label style={{ display: 'block', fontSize: '12px', color: C.textSec, marginBottom: '4px' }}>Max Tokens</label>
                              <input
                                type="number" step="1000"
                                value={sceneForm.max_tokens ?? ''}
                                onChange={e => setSceneForm(p => ({ ...p, max_tokens: e.target.value === '' ? null : parseInt(e.target.value) }))}
                                placeholder="继承全局"
                                style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', boxSizing: 'border-box' }}
                                onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                                onBlur={e => { e.currentTarget.style.borderColor = C.border }}
                              />
                            </div>
                            <div>
                              <label style={{ display: 'block', fontSize: '12px', color: C.textSec, marginBottom: '4px' }}>状态</label>
                              <select
                                value={sceneForm.is_active ? 'true' : 'false'}
                                onChange={e => setSceneForm(p => ({ ...p, is_active: e.target.value === 'true' }))}
                                style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', boxSizing: 'border-box', background: C.white }}
                              >
                                <option value="true">启用</option>
                                <option value="false">禁用</option>
                              </select>
                            </div>
                          </div>
                          <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
                            <button
                              onClick={() => setEditingScene(null)}
                              style={{ padding: '7px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '13px', color: C.textSec, cursor: 'pointer' }}
                            >
                              取消
                            </button>
                            <button
                              onClick={() => handleSaveScene(scene.scene_code)}
                              disabled={sceneSaving}
                              style={{
                                padding: '7px 20px', borderRadius: '8px', border: 'none',
                                background: `linear-gradient(135deg,${C.primary},#7C3AED)`,
                                color: '#fff', fontSize: '13px', fontWeight: 600,
                                cursor: 'pointer', opacity: sceneSaving ? 0.6 : 1,
                              }}
                            >
                              {sceneSaving ? '保存中...' : '保存'}
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
