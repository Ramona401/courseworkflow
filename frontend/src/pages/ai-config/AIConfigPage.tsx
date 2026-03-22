/**
 * AI配置中心页面（仅admin可访问）
 * - 全局配置区：API地址、Key、模型（自由填写，不限制选项）、温度、Token数
 * - AI连通性测试：测试按钮+结果显示（P2-2新增）
 * - 场景配置区：6个Pipeline步骤各自的AI参数（模型自由填写）
 * - Apple 风格：毛玻璃卡片 + 渐变 + 圆角 + 微动效
 */
import { useState, useEffect, useCallback } from 'react'
import { getGlobalConfig, updateGlobalConfig, getSceneConfigs, updateSceneConfig, testConnection } from '@/api/ai-config'
import type { GlobalConfig, SceneConfig, UpdateSceneConfigRequest, TestConnectionResult } from '@/api/ai-config'
import { Bot, Save, Eye, EyeOff, RefreshCw, Zap, Wifi, WifiOff, Clock } from 'lucide-react'

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
  onFocus: (e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) => {
    e.currentTarget.style.borderColor = '#007aff'
    e.currentTarget.style.boxShadow = '0 0 0 3px rgba(0,122,255,0.12)'
  },
  onBlur: (e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) => {
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

const selectStyle: React.CSSProperties = {
  ...inputStyle,
  appearance: 'none' as const,
  backgroundImage: 'url("data:image/svg+xml,%3Csvg xmlns=\'http://www.w3.org/2000/svg\' width=\'12\' height=\'12\' viewBox=\'0 0 12 12\'%3E%3Cpath d=\'M6 8L1 3h10z\' fill=\'%238e8e93\'/%3E%3C/svg%3E")',
  backgroundRepeat: 'no-repeat',
  backgroundPosition: 'right 12px center',
  paddingRight: '36px',
}

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

export default function AIConfigPage() {
  // 全局配置状态
  const [globalConfig, setGlobalConfig] = useState<GlobalConfig | null>(null)
  const [globalForm, setGlobalForm] = useState({
    api_base_url: '',
    api_key: '',
    default_model: '',
    temperature: '',
    max_tokens: '',
  })
  const [showKey, setShowKey] = useState(false)
  const [globalSaving, setGlobalSaving] = useState(false)

  // AI连通性测试状态（P2-2新增）
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<TestConnectionResult | null>(null)

  // 场景配置状态
  const [scenes, setScenes] = useState<SceneConfig[]>([])
  const [editingScene, setEditingScene] = useState<string | null>(null)
  const [sceneForm, setSceneForm] = useState<UpdateSceneConfigRequest>({})
  const [sceneSaving, setSceneSaving] = useState(false)

  // 通用状态
  const [loading, setLoading] = useState(true)
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)

  // ==================== 数据加载 ====================

  const loadData = useCallback(async () => {
    try {
      setLoading(true)
      const [gc, sc] = await Promise.all([getGlobalConfig(), getSceneConfigs()])
      setGlobalConfig(gc)
      setGlobalForm({
        api_base_url: gc.api_base_url || '',
        api_key: '',
        default_model: gc.default_model || '',
        temperature: gc.temperature || '',
        max_tokens: gc.max_tokens || '',
      })
      setScenes(sc || [])
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '加载失败'
      setToast({ message: msg, type: 'error' })
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { loadData() }, [loadData])

  // ==================== 全局配置保存 ====================

  const handleSaveGlobal = async () => {
    try {
      setGlobalSaving(true)
      const result = await updateGlobalConfig(globalForm)
      setGlobalConfig(result)
      setGlobalForm(prev => ({ ...prev, api_key: '' }))
      setToast({ message: '全局配置保存成功', type: 'success' })
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '保存失败'
      setToast({ message: msg, type: 'error' })
    } finally {
      setGlobalSaving(false)
    }
  }

  // ==================== AI连通性测试（P2-2新增）====================

  const handleTestConnection = async () => {
    try {
      setTesting(true)
      setTestResult(null)
      const result = await testConnection()
      setTestResult(result)
      if (result.success) {
        setToast({ message: '连接测试成功！', type: 'success' })
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '测试请求失败'
      setTestResult({
        success: false,
        message: msg,
        latency_ms: 0,
        model: '',
        api_base_url: '',
      })
    } finally {
      setTesting(false)
    }
  }

  // ==================== 场景配置编辑 ====================

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
      setToast({ message: `场景 "${code}" 配置保存成功`, type: 'success' })
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '保存失败'
      setToast({ message: msg, type: 'error' })
    } finally {
      setSceneSaving(false)
    }
  }

  const handleCancelScene = () => {
    setEditingScene(null)
    setSceneForm({})
  }

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
          <div style={{ color: '#8e8e93', fontSize: '14px' }}>加载AI配置...</div>
        </div>
      </div>
    )
  }

  // ==================== 渲染 ====================

  return (
    <div style={{ maxWidth: '960px', margin: '0 auto' }}>
      {/* Toast 提示 */}
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      {/* 页面标题 */}
      <div style={{ marginBottom: '28px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '8px' }}>
          <div style={{
            width: '40px', height: '40px',
            background: 'linear-gradient(135deg, #007aff, #5856d6)',
            borderRadius: '12px',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            boxShadow: '0 2px 8px rgba(0,122,255,0.3)',
          }}>
            <Bot size={22} color="#fff" />
          </div>
          <div>
            <h1 style={{ fontSize: '22px', fontWeight: 700, color: '#1d1d1f', margin: 0 }}>AI 配置中心</h1>
            <p style={{ fontSize: '13px', color: '#8e8e93', margin: '2px 0 0 0' }}>管理 AI API 连接和各场景参数</p>
          </div>
        </div>
      </div>

      {/* ==================== 全局配置卡片 ==================== */}
      <div style={cardStyle}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '24px' }}>
          <h2 style={{ fontSize: '17px', fontWeight: 600, color: '#1d1d1f', margin: 0 }}>全局配置</h2>
          <div style={{
            padding: '4px 10px', borderRadius: '6px', fontSize: '12px', fontWeight: 500,
            background: globalConfig?.api_key_set ? 'rgba(52,199,89,0.12)' : 'rgba(255,59,48,0.12)',
            color: globalConfig?.api_key_set ? '#34c759' : '#ff3b30',
          }}>
            {globalConfig?.api_key_set ? 'API Key 已配置' : 'API Key 未配置'}
          </div>
        </div>

        {/* API 基础地址 */}
        <div style={{ marginBottom: '18px' }}>
          <label style={labelStyle}>API 基础地址</label>
          <input
            style={inputStyle}
            value={globalForm.api_base_url}
            onChange={e => setGlobalForm(prev => ({ ...prev, api_base_url: e.target.value }))}
            placeholder="https://api.openai.com/v1"
            {...inputFocusProps}
          />
        </div>

        {/* API Key */}
        <div style={{ marginBottom: '18px' }}>
          <label style={labelStyle}>
            API Key
            {globalConfig?.api_key_set && (
              <span style={{ fontWeight: 400, color: '#8e8e93', marginLeft: '8px' }}>
                当前：{globalConfig.api_key}
              </span>
            )}
          </label>
          <div style={{ position: 'relative' }}>
            <input
              style={{ ...inputStyle, paddingRight: '44px' }}
              type={showKey ? 'text' : 'password'}
              value={globalForm.api_key}
              onChange={e => setGlobalForm(prev => ({ ...prev, api_key: e.target.value }))}
              placeholder={globalConfig?.api_key_set ? '留空表示不修改' : '请输入 API Key'}
              {...inputFocusProps}
            />
            <button
              onClick={() => setShowKey(!showKey)}
              style={{
                position: 'absolute', right: '10px', top: '50%', transform: 'translateY(-50%)',
                background: 'none', border: 'none', cursor: 'pointer', color: '#8e8e93',
                padding: '4px', display: 'flex',
              }}
            >
              {showKey ? <EyeOff size={16} /> : <Eye size={16} />}
            </button>
          </div>
        </div>

        {/* 默认模型（自由填写）+ 温度 + Token数（三列） */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '16px', marginBottom: '24px' }}>
          <div>
            {/* 模型改为自由填写文本框，不限制可选模型范围 */}
            <label style={labelStyle}>默认模型</label>
            <input
              style={inputStyle}
              type="text"
              value={globalForm.default_model}
              onChange={e => setGlobalForm(prev => ({ ...prev, default_model: e.target.value }))}
              placeholder="如 anthropic/claude-haiku-4.5"
              {...inputFocusProps}
            />
          </div>
          <div>
            <label style={labelStyle}>默认温度</label>
            <input
              style={inputStyle}
              type="number"
              step="0.1"
              min="0"
              max="2"
              value={globalForm.temperature}
              onChange={e => setGlobalForm(prev => ({ ...prev, temperature: e.target.value }))}
              {...inputFocusProps}
            />
          </div>
          <div>
            <label style={labelStyle}>最大 Token 数</label>
            <input
              style={inputStyle}
              type="number"
              step="1000"
              min="100"
              max="200000"
              value={globalForm.max_tokens}
              onChange={e => setGlobalForm(prev => ({ ...prev, max_tokens: e.target.value }))}
              {...inputFocusProps}
            />
          </div>
        </div>

        {/* 操作按钮行：重置 + 测试连接 + 保存 */}
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px', alignItems: 'center' }}>
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
          {/* 测试连接按钮（P2-2新增） */}
          <button
            onClick={handleTestConnection}
            disabled={testing}
            style={{
              ...btnPrimary,
              background: testing
                ? 'linear-gradient(135deg, #8e8e93, #aeaeb2)'
                : 'linear-gradient(135deg, #ff9500, #ff6b00)',
              boxShadow: testing ? 'none' : '0 2px 8px rgba(255,149,0,0.3)',
              opacity: testing ? 0.7 : 1,
            }}
            onMouseEnter={e => { if (!testing) (e.currentTarget as HTMLElement).style.transform = 'translateY(-1px)' }}
            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.transform = 'none' }}
          >
            <Wifi size={15} />
            {testing ? '测试中...' : '测试连接'}
          </button>
          <button
            onClick={handleSaveGlobal}
            disabled={globalSaving}
            style={{ ...btnPrimary, opacity: globalSaving ? 0.6 : 1 }}
            onMouseEnter={e => { if (!globalSaving) (e.currentTarget as HTMLElement).style.transform = 'translateY(-1px)' }}
            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.transform = 'none' }}
          >
            <Save size={15} />
            {globalSaving ? '保存中...' : '保存全局配置'}
          </button>
        </div>

        {/* AI连通性测试结果（P2-2新增） */}
        {testResult && (
          <div style={{
            marginTop: '20px',
            padding: '16px 20px',
            borderRadius: '12px',
            border: testResult.success
              ? '1px solid rgba(52,199,89,0.3)'
              : '1px solid rgba(255,59,48,0.3)',
            background: testResult.success
              ? 'rgba(52,199,89,0.06)'
              : 'rgba(255,59,48,0.06)',
          }}>
            {/* 结果标题行 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '10px' }}>
              {testResult.success ? (
                <Wifi size={18} color="#34c759" />
              ) : (
                <WifiOff size={18} color="#ff3b30" />
              )}
              <span style={{
                fontSize: '15px',
                fontWeight: 600,
                color: testResult.success ? '#34c759' : '#ff3b30',
              }}>
                {testResult.success ? '连接成功' : '连接失败'}
              </span>
              {testResult.latency_ms > 0 && (
                <span style={{
                  display: 'inline-flex', alignItems: 'center', gap: '4px',
                  fontSize: '12px', color: '#8e8e93',
                  padding: '2px 8px', borderRadius: '6px',
                  background: 'rgba(0,0,0,0.04)',
                }}>
                  <Clock size={12} />
                  {testResult.latency_ms}ms
                </span>
              )}
            </div>
            {/* 详细信息 */}
            <div style={{ fontSize: '13px', color: '#1d1d1f', lineHeight: '1.6' }}>
              <div>{testResult.message}</div>
              {testResult.model && (
                <div style={{ color: '#8e8e93', marginTop: '4px' }}>
                  模型：{testResult.model}
                </div>
              )}
              {testResult.api_base_url && (
                <div style={{ color: '#8e8e93' }}>
                  地址：{testResult.api_base_url}
                </div>
              )}
            </div>
            {/* 关闭按钮 */}
            <button
              onClick={() => setTestResult(null)}
              style={{
                marginTop: '10px',
                padding: '4px 12px', borderRadius: '6px',
                border: '1px solid #d2d2d7', background: '#fff',
                fontSize: '12px', color: '#8e8e93', cursor: 'pointer',
              }}
            >
              关闭
            </button>
          </div>
        )}
      </div>

      {/* ==================== 场景配置卡片 ==================== */}
      <div style={cardStyle}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '24px' }}>
          <Zap size={18} color="#ff9500" />
          <h2 style={{ fontSize: '17px', fontWeight: 600, color: '#1d1d1f', margin: 0 }}>场景配置</h2>
          <span style={{ fontSize: '12px', color: '#8e8e93' }}>（每个场景可独立覆盖全局配置，留空则继承全局）</span>
        </div>

        {/* 场景列表 */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {scenes.map(scene => {
            const isEditing = editingScene === scene.scene_code

            return (
              <div key={scene.scene_code} style={{
                padding: '16px 20px',
                borderRadius: '12px',
                border: isEditing ? '1px solid #007aff' : '1px solid rgba(0,0,0,0.06)',
                background: isEditing ? 'rgba(0,122,255,0.03)' : 'rgba(0,0,0,0.015)',
                transition: 'all 0.2s',
              }}>
                {/* 场景头部 */}
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: isEditing ? '16px' : '0' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <div style={{
                      width: '8px', height: '8px', borderRadius: '50%',
                      background: scene.is_active ? '#34c759' : '#8e8e93',
                    }} />
                    <span style={{ fontSize: '15px', fontWeight: 600, color: '#1d1d1f' }}>{scene.scene_name}</span>
                    <span style={{ fontSize: '12px', color: '#8e8e93', fontFamily: 'monospace' }}>{scene.scene_code}</span>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    {!isEditing && (
                      <>
                        <span style={{ fontSize: '12px', color: '#8e8e93' }}>
                          {scene.model || '继承全局'} · T:{scene.temperature ?? '继承'} · Max:{scene.max_tokens ?? '继承'}
                        </span>
                        <button
                          onClick={() => handleEditScene(scene)}
                          style={{
                            padding: '5px 14px', borderRadius: '8px', border: '1px solid #d2d2d7',
                            background: '#fff', fontSize: '12px', fontWeight: 500, color: '#007aff',
                            cursor: 'pointer', transition: 'all 0.15s',
                          }}
                          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = '#f0f0f5' }}
                          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = '#fff' }}
                        >
                          编辑
                        </button>
                      </>
                    )}
                  </div>
                </div>

                {/* 编辑表单（展开） */}
                {isEditing && (
                  <div>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 120px', gap: '12px', marginBottom: '16px' }}>
                      <div>
                        {/* 场景模型改为自由填写，留空则继承全局配置 */}
                        <label style={{ ...labelStyle, fontSize: '12px' }}>模型</label>
                        <input
                          style={{ ...inputStyle, padding: '8px 12px', fontSize: '13px' }}
                          type="text"
                          value={sceneForm.model || ''}
                          onChange={e => setSceneForm(prev => ({ ...prev, model: e.target.value || null }))}
                          placeholder="留空继承全局"
                          {...inputFocusProps}
                        />
                      </div>
                      <div>
                        <label style={{ ...labelStyle, fontSize: '12px' }}>温度</label>
                        <input
                          style={{ ...inputStyle, padding: '8px 12px', fontSize: '13px' }}
                          type="number"
                          step="0.1"
                          min="0"
                          max="2"
                          value={sceneForm.temperature ?? ''}
                          onChange={e => setSceneForm(prev => ({
                            ...prev,
                            temperature: e.target.value === '' ? null : parseFloat(e.target.value),
                          }))}
                          placeholder="继承全局"
                          {...inputFocusProps}
                        />
                      </div>
                      <div>
                        <label style={{ ...labelStyle, fontSize: '12px' }}>最大 Token</label>
                        <input
                          style={{ ...inputStyle, padding: '8px 12px', fontSize: '13px' }}
                          type="number"
                          step="1000"
                          min="100"
                          max="200000"
                          value={sceneForm.max_tokens ?? ''}
                          onChange={e => setSceneForm(prev => ({
                            ...prev,
                            max_tokens: e.target.value === '' ? null : parseInt(e.target.value),
                          }))}
                          placeholder="继承全局"
                          {...inputFocusProps}
                        />
                      </div>
                      <div>
                        <label style={{ ...labelStyle, fontSize: '12px' }}>状态</label>
                        <select
                          style={{ ...selectStyle, padding: '8px 12px', fontSize: '13px' }}
                          value={sceneForm.is_active ? 'true' : 'false'}
                          onChange={e => setSceneForm(prev => ({ ...prev, is_active: e.target.value === 'true' }))}
                          {...inputFocusProps}
                        >
                          <option value="true">启用</option>
                          <option value="false">禁用</option>
                        </select>
                      </div>
                    </div>
                    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                      <button
                        onClick={handleCancelScene}
                        style={{
                          padding: '6px 16px', borderRadius: '8px', border: '1px solid #d2d2d7',
                          background: '#fff', fontSize: '13px', color: '#1d1d1f', cursor: 'pointer',
                        }}
                      >
                        取消
                      </button>
                      <button
                        onClick={() => handleSaveScene(scene.scene_code)}
                        disabled={sceneSaving}
                        style={{
                          padding: '6px 16px', borderRadius: '8px', border: 'none',
                          background: 'linear-gradient(135deg, #007aff, #5856d6)',
                          fontSize: '13px', fontWeight: 600, color: '#fff', cursor: 'pointer',
                          opacity: sceneSaving ? 0.6 : 1,
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
    </div>
  )
}
