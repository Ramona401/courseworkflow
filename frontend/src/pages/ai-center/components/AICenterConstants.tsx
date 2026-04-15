/* eslint-disable react-refresh/only-export-components */
/**
 * AICenterConstants — AI管理中心共享常量+小组件
 *
 * v89-3拆分：从AICenterPage.tsx中拆出
 * 包含：样式常量、供应商目录、Toast、ProviderItem、ModelSelect
 */
import { useState, useEffect } from 'react'

// ==================== 供应商类型 ====================

export interface AIProvider {
  id: string
  name: string
  logo: string
  baseURL: string
  docsURL: string
  models: string[]
  description: string
  isChinese: boolean
}

// ==================== 供应商目录 ====================

export const AI_PROVIDERS: AIProvider[] = [
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

export const C = {
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

// ==================== Toast组件 ====================

export function Toast({ message, type, onClose }: {
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

export function ProviderItem({ provider, selected, onSelect }: {
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

export function ModelSelect({
  value,
  onChange,
  availableModels,
  placeholder,
}: {
  value: string | null
  onChange: (v: string | null) => void
  availableModels: string[]
  placeholder?: string
}) {
  const isCustom = value !== null && value !== '' && !availableModels.includes(value)
  const [showCustomInput, setShowCustomInput] = useState(isCustom)
  const [customVal, setCustomVal] = useState(isCustom ? value : '')

  const handleSelectChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const v = e.target.value
    if (v === '__custom__') {
      setShowCustomInput(true)
      setCustomVal('')
      onChange(null)
    } else if (v === '__inherit__') {
      setShowCustomInput(false)
      setCustomVal('')
      onChange(null)
    } else {
      setShowCustomInput(false)
      setCustomVal('')
      onChange(v)
    }
  }

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
          outline: 'none', background: C.white, cursor: 'pointer', color: C.text,
        }}
        onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
        onBlur={e => { e.currentTarget.style.borderColor = C.border }}
      >
        <option value="__inherit__">— 继承全局配置 —</option>
        {availableModels.length > 0 && (
          <optgroup label={`可用模型（${availableModels.length}个）`}>
            {availableModels.map(m => (
              <option key={m} value={m}>{m}</option>
            ))}
          </optgroup>
        )}
        <option value="__custom__">✏️ 手动输入模型名...</option>
      </select>
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
