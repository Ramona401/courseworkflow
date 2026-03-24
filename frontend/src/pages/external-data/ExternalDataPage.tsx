/**
 * 外部数据配置页面（P3-1新增，仅admin可访问）
 * - OSS配置区：Endpoint、Bucket、AccessKey ID、AccessKey Secret、索引前缀、HTML前缀
 * - 推送API配置区：推送地址、推送Token
 * - 敏感字段（AccessKey Secret、推送Token）加密存储，脱敏显示
 * - Apple 风格：毛玻璃卡片 + 渐变 + 圆角 + 微动效
 */
import { useState, useEffect, useCallback } from 'react'
import { getExternalDataConfigs, updateExternalDataConfigs } from '@/api/external-data'
import type { ExternalDataConfigItem } from '@/api/external-data'
import { Database, Save, RefreshCw, Eye, EyeOff, Cloud, Send } from 'lucide-react'

// ==================== 样式常量 ====================

const cardStyle: React.CSSProperties = {
  background: 'rgba(255,255,255,0.72)',
  backdropFilter: 'blur(20px)',
  WebkitBackdropFilter: 'blur(20px)',
  borderRadius: '16px',
  border: '1px solid rgba(0,0,0,0.06)',
  padding: '28px',
  marginBottom: '24px',
  boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
}

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: '13px',
  fontWeight: 600,
  color: '#1d1d1f',
  marginBottom: '6px',
}

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '10px 14px',
  borderRadius: '10px',
  border: '1px solid #d2d2d7',
  fontSize: '14px',
  color: '#1d1d1f',
  background: '#fff',
  outline: 'none',
  transition: 'border-color 0.2s, box-shadow 0.2s',
  boxSizing: 'border-box' as const,
}

const inputFocusProps = {
  onFocus: (e: React.FocusEvent<HTMLInputElement>) => {
    e.currentTarget.style.borderColor = '#007aff'
    e.currentTarget.style.boxShadow = '0 0 0 3px rgba(0,122,255,0.12)'
  },
  onBlur: (e: React.FocusEvent<HTMLInputElement>) => {
    e.currentTarget.style.borderColor = '#d2d2d7'
    e.currentTarget.style.boxShadow = 'none'
  },
}

const btnPrimary: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: '8px',
  padding: '10px 24px',
  borderRadius: '10px',
  border: 'none',
  fontSize: '14px',
  fontWeight: 600,
  color: '#fff',
  background: 'linear-gradient(135deg, #007aff, #5856d6)',
  cursor: 'pointer',
  transition: 'all 0.2s',
  boxShadow: '0 2px 8px rgba(0,122,255,0.25)',
}

// ==================== 配置项定义 ====================

/** OSS配置项元数据 */
const OSS_FIELDS = [
  { key: 'oss_endpoint', label: 'OSS Endpoint', placeholder: '如 oss-cn-beijing.aliyuncs.com', sensitive: false },
  { key: 'oss_bucket', label: 'OSS Bucket', placeholder: '如 my-bucket-name', sensitive: false },
  { key: 'oss_access_key_id', label: 'AccessKey ID', placeholder: '阿里云 AccessKey ID', sensitive: false },
  { key: 'oss_access_key_enc', label: 'AccessKey Secret', placeholder: '阿里云 AccessKey Secret（加密存储）', sensitive: true },
  { key: 'oss_index_prefix', label: '索引文件路径前缀', placeholder: '如 indexes/', sensitive: false },
  { key: 'oss_html_prefix', label: 'HTML文件路径前缀', placeholder: '如 lessons/', sensitive: false },
]

/** 推送API配置项元数据 */
const PUSH_FIELDS = [
  { key: 'push_api_url', label: '推送API地址', placeholder: '推送回原始服务器的API地址', sensitive: false },
  { key: 'push_api_token', label: '推送API Token', placeholder: '推送认证Token（加密存储）', sensitive: true },
]

// ==================== Toast 组件 ====================

function Toast({ message, type, onClose }: { message: string; type: 'success' | 'error'; onClose: () => void }) {
  useEffect(() => {
    const timer = setTimeout(onClose, 3000)
    return () => clearTimeout(timer)
  }, [onClose])

  return (
    <div style={{
      position: 'fixed',
      top: '24px',
      right: '24px',
      padding: '12px 20px',
      borderRadius: '12px',
      color: '#fff',
      fontSize: '14px',
      fontWeight: 500,
      background: type === 'success'
        ? 'linear-gradient(135deg, #34c759, #30d158)'
        : 'linear-gradient(135deg, #ff3b30, #ff453a)',
      boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
      zIndex: 9999,
      animation: 'slideIn 0.3s ease',
    }}>
      {message}
    </div>
  )
}

// ==================== 主组件 ====================

export default function ExternalDataPage() {
  // 配置数据状态
  const [configs, setConfigs] = useState<ExternalDataConfigItem[]>([])
  const [formValues, setFormValues] = useState<Record<string, string>>({})
  const [showSensitive, setShowSensitive] = useState<Record<string, boolean>>({})

  // 通用状态
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)

  // ==================== 数据加载 ====================

  const loadData = useCallback(async () => {
    try {
      setLoading(true)
      const result = await getExternalDataConfigs()
      setConfigs(result.configs || [])

      // 初始化表单值：非敏感字段用实际值，敏感字段留空（空表示不修改）
      const initialValues: Record<string, string> = {}
      for (const c of (result.configs || [])) {
        if (c.is_sensitive) {
          initialValues[c.config_key] = '' // 敏感字段：空=不修改
        } else {
          initialValues[c.config_key] = c.config_value || ''
        }
      }
      setFormValues(initialValues)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '加载失败'
      setToast({ message: msg, type: 'error' })
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { loadData() }, [loadData])

  // ==================== 保存配置 ====================

  const handleSave = async () => {
    try {
      setSaving(true)
      const result = await updateExternalDataConfigs({ configs: formValues })
      setConfigs(result.configs || [])

      // 重置敏感字段输入框为空
      const resetValues: Record<string, string> = {}
      for (const c of (result.configs || [])) {
        if (c.is_sensitive) {
          resetValues[c.config_key] = ''
        } else {
          resetValues[c.config_key] = c.config_value || ''
        }
      }
      setFormValues(resetValues)

      setToast({ message: '外部数据配置保存成功', type: 'success' })
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '保存失败'
      setToast({ message: msg, type: 'error' })
    } finally {
      setSaving(false)
    }
  }

  // ==================== 工具方法 ====================

  /** 获取配置项的当前状态信息 */
  const getConfigInfo = (key: string): ExternalDataConfigItem | undefined => {
    return configs.find(c => c.config_key === key)
  }

  /** 切换敏感字段可见性 */
  const toggleSensitive = (key: string) => {
    setShowSensitive(prev => ({ ...prev, [key]: !prev[key] }))
  }

  /** 渲染单个配置输入项 */
  const renderField = (field: { key: string; label: string; placeholder: string; sensitive: boolean }) => {
    const info = getConfigInfo(field.key)
    const value = formValues[field.key] || ''

    return (
      <div key={field.key} style={{ marginBottom: '18px' }}>
        <label style={labelStyle}>
          {field.label}
          {field.sensitive && info && (
            <span style={{
              fontWeight: 400,
              color: '#8e8e93',
              marginLeft: '8px',
              fontSize: '12px',
            }}>
              {info.is_set ? '当前：' + info.config_value : '未配置'}
            </span>
          )}
          {!field.sensitive && info && !info.is_set && (
            <span style={{
              fontWeight: 400,
              color: '#ff9500',
              marginLeft: '8px',
              fontSize: '12px',
            }}>
              待配置
            </span>
          )}
        </label>
        <div style={{ position: 'relative' }}>
          <input
            style={{
              ...inputStyle,
              paddingRight: field.sensitive ? '44px' : '14px',
            }}
            type={field.sensitive && !showSensitive[field.key] ? 'password' : 'text'}
            value={value}
            onChange={e => setFormValues(prev => ({ ...prev, [field.key]: e.target.value }))}
            placeholder={field.sensitive && info?.is_set
              ? '留空表示不修改'
              : field.placeholder}
            {...inputFocusProps}
          />
          {field.sensitive && (
            <button
              onClick={() => toggleSensitive(field.key)}
              style={{
                position: 'absolute',
                right: '10px',
                top: '50%',
                transform: 'translateY(-50%)',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                color: '#8e8e93',
                padding: '4px',
                display: 'flex',
              }}
            >
              {showSensitive[field.key] ? <EyeOff size={16} /> : <Eye size={16} />}
            </button>
          )}
        </div>
      </div>
    )
  }

  // ==================== 统计信息 ====================

  const totalConfigs = configs.length
  const configuredCount = configs.filter(c => c.is_set).length
  const pendingCount = totalConfigs - configuredCount

  // ==================== 加载状态 ====================

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '60vh' }}>
        <div style={{ textAlign: 'center' }}>
          <div style={{
            width: '32px', height: '32px',
            border: '2px solid #007aff', borderTopColor: 'transparent',
            borderRadius: '50%', animation: 'spin 0.8s linear infinite',
            margin: '0 auto 12px',
          }} />
          <div style={{ color: '#8e8e93', fontSize: '14px' }}>加载外部数据配置...</div>
        </div>
      </div>
    )
  }

  // ==================== 渲染 ====================

  return (
    <div style={{ maxWidth: '960px', margin: '0 auto' }}>
      {/* Toast 提示 */}
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      {/* 页面描述（标题由MainLayout header提供） */}
      <p style={{ fontSize: '14px', color: '#8e8e93', margin: '0 0 20px 0' }}>配置 OSS 数据源和推送 API 连接信息</p>

      {/* 统计卡片 */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '16px', marginBottom: '24px' }}>
        {[
          { label: '总配置项', value: totalConfigs, color: '#007aff' },
          { label: '已配置', value: configuredCount, color: '#34c759' },
          { label: '待配置', value: pendingCount, color: '#ff9500' },
        ].map(stat => (
          <div key={stat.label} style={{
            ...cardStyle,
            marginBottom: 0,
            padding: '20px',
            textAlign: 'center',
          }}>
            <div style={{ fontSize: '28px', fontWeight: 700, color: stat.color }}>{stat.value}</div>
            <div style={{ fontSize: '13px', color: '#8e8e93', marginTop: '4px' }}>{stat.label}</div>
          </div>
        ))}
      </div>

      {/* ==================== OSS 配置卡片 ==================== */}
      <div style={cardStyle}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '24px' }}>
          <Cloud size={18} color="#007aff" />
          <h2 style={{ fontSize: '17px', fontWeight: 600, color: '#1d1d1f', margin: 0 }}>OSS 数据源配置</h2>
          <span style={{ fontSize: '12px', color: '#8e8e93' }}>（阿里云 OSS 只读拉取课程索引和 HTML）</span>
        </div>

        {/* 第一行：Endpoint + Bucket（两列） */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
          {OSS_FIELDS.slice(0, 2).map(field => renderField(field))}
        </div>

        {/* 第二行：AccessKey ID + AccessKey Secret（两列） */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
          {OSS_FIELDS.slice(2, 4).map(field => renderField(field))}
        </div>

        {/* 第三行：索引前缀 + HTML前缀（两列） */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
          {OSS_FIELDS.slice(4, 6).map(field => renderField(field))}
        </div>
      </div>

      {/* ==================== 推送API 配置卡片 ==================== */}
      <div style={cardStyle}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '24px' }}>
          <Send size={18} color="#ff9500" />
          <h2 style={{ fontSize: '17px', fontWeight: 600, color: '#1d1d1f', margin: 0 }}>推送 API 配置</h2>
          <span style={{ fontSize: '12px', color: '#8e8e93' }}>（定稿后推送回原始服务器）</span>
        </div>

        {PUSH_FIELDS.map(field => renderField(field))}
      </div>

      {/* ==================== 操作按钮 ==================== */}
      <div style={{
        ...cardStyle,
        display: 'flex',
        justifyContent: 'flex-end',
        gap: '12px',
        alignItems: 'center',
        padding: '20px 28px',
      }}>
        <button
          onClick={loadData}
          style={{
            ...btnPrimary,
            background: '#f5f5f7',
            color: '#1d1d1f',
            boxShadow: 'none',
          }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = '#e8e8ed' }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = '#f5f5f7' }}
        >
          <RefreshCw size={15} />
          重置
        </button>
        <button
          onClick={handleSave}
          disabled={saving}
          style={{ ...btnPrimary, opacity: saving ? 0.6 : 1 }}
          onMouseEnter={e => { if (!saving) (e.currentTarget as HTMLElement).style.transform = 'translateY(-1px)' }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.transform = 'none' }}
        >
          <Save size={15} />
          {saving ? '保存中...' : '保存所有配置'}
        </button>
      </div>

      {/* ==================== 使用说明 ==================== */}
      <div style={{
        ...cardStyle,
        background: 'rgba(0,122,255,0.03)',
        border: '1px solid rgba(0,122,255,0.1)',
      }}>
        <h3 style={{ fontSize: '14px', fontWeight: 600, color: '#007aff', margin: '0 0 12px 0' }}>使用说明</h3>
        <div style={{ fontSize: '13px', color: '#6e6e73', lineHeight: '1.8' }}>
          <p style={{ margin: '0 0 8px 0' }}>
            <strong>OSS 配置：</strong>用于从阿里云 OSS 只读拉取课程索引和原始 HTML 文件。
            AccessKey 建议使用只读权限的子账号密钥。AccessKey Secret 会使用 AES-256 加密后存储。
          </p>
          <p style={{ margin: '0 0 8px 0' }}>
            <strong>推送 API 配置：</strong>Pipeline 定稿后，将修改后的课程内容推送回原始服务器。
            推送 Token 同样会加密存储。
          </p>
          <p style={{ margin: 0 }}>
            <strong>安全提示：</strong>敏感字段（AccessKey Secret、推送 Token）保存后仅显示脱敏值，
            留空提交表示不修改该字段。配置信息变更会记录操作者和时间。
          </p>
        </div>
      </div>
    </div>
  )
}
