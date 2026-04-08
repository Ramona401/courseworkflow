/**
 * AICenterPage — 统一 AI API 管理中心（admin专属）
 *
 * v89-3拆分：
 *   - 常量+小组件 → components/AICenterConstants.tsx
 *   - 降级模型选择器 → components/FallbackModelsPicker.tsx
 *   - 场景配置Tab → components/SceneConfigPanel.tsx
 *   - 本文件保留：主页面框架 + 状态管理 + 连接配置Tab
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  getGlobalConfig, updateGlobalConfig,
  getSceneConfigs, updateSceneConfig,
  testConnection, listModels,
} from '@/api/ai-config'
import type { GlobalConfig, SceneConfig, UpdateSceneConfigRequest } from '@/api/ai-config'
import { AI_PROVIDERS, C, Toast, ProviderItem, ModelSelect } from './components/AICenterConstants'
import type { AIProvider } from './components/AICenterConstants'
import SceneConfigPanel from './components/SceneConfigPanel'

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

  // 可用模型列表
  const [modelsLoading, setModelsLoading] = useState(false)
  const [availableModels, setAvailableModels] = useState<string[]>([])
  const [modelsError, setModelsError] = useState<string | null>(null)
  const [modelsQueried, setModelsQueried] = useState(false)

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

  // ==================== 事件处理 ====================

  const handleSelectProvider = (p: AIProvider) => {
    setSelectedProvider(p)
    setForm(prev => ({ ...prev, api_base_url: p.baseURL }))
    setTestResult(null)
    setAvailableModels([])
    setModelsQueried(false)
    setModelsError(null)
  }

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

  const handleListModels = async () => {
    try {
      setModelsLoading(true)
      setModelsError(null)
      const r = await listModels()
      const ids = (r.models || []).map(m => m.id)
      setAvailableModels(ids)
      setModelsQueried(true)
      showToast(`查询成功，共 ${ids.length} 个可用模型`, 'success')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '查询失败'
      setModelsError(msg)
      setAvailableModels([])
      setModelsQueried(true)
    } finally {
      setModelsLoading(false)
    }
  }

  // 场景配置处理
  const handleEditScene = (scene: SceneConfig) => {
    setEditingScene(scene.scene_code)
    setSceneForm({
      model: scene.model,
      temperature: scene.temperature,
      max_tokens: scene.max_tokens,
      is_active: scene.is_active,
      fallback_models: scene.fallback_models || [],
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
        <button onClick={() => navigate(fromPath)} style={{
          display: 'flex', alignItems: 'center', gap: '6px',
          padding: '8px 16px', borderRadius: '8px',
          border: `1px solid ${C.border}`, background: C.white,
          fontSize: '14px', color: C.textSec, cursor: 'pointer',
        }}>{'<- 返回'}</button>
        <div style={{ flex: 1, textAlign: 'center' }}>
          <h1 style={{ fontSize: '18px', fontWeight: 700, color: C.text, margin: 0 }}>🤖 AI API 管理中心</h1>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>统一管理大模型供应商与场景配置</div>
        </div>
        <div style={{
          padding: '6px 14px', borderRadius: '20px', fontSize: '12px', fontWeight: 600,
          background: globalConfig?.api_key_set ? 'rgba(16,185,129,0.1)' : 'rgba(239,68,68,0.1)',
          color: globalConfig?.api_key_set ? C.success : C.danger,
        }}>
          {globalConfig?.api_key_set ? '✓ API Key 已配置' : '⚠ API Key 未配置'}
        </div>
      </header>

      {/* 主体 */}
      <div style={{ display: 'flex', maxWidth: '1280px', margin: '0 auto', padding: '28px 24px', gap: '24px', alignItems: 'flex-start' }}>

        {/* 左侧供应商选择器 */}
        <aside style={{
          width: '260px', flexShrink: 0, background: C.card, borderRadius: '16px',
          border: `1px solid ${C.border}`, boxShadow: '0 2px 8px rgba(0,0,0,0.04)',
          overflow: 'hidden', position: 'sticky', top: '88px',
        }}>
          <div style={{ padding: '14px 20px', borderBottom: `1px solid ${C.border}`, fontSize: '13px', fontWeight: 600, color: C.textSec }}>选择供应商</div>
          <div style={{ padding: '8px 10px' }}>
            <div style={{ fontSize: '11px', color: C.textMuted, padding: '4px 8px', fontWeight: 600 }}>🌍 国际</div>
            {AI_PROVIDERS.filter(p => !p.isChinese).map(p => (
              <ProviderItem key={p.id} provider={p} selected={selectedProvider.id === p.id} onSelect={handleSelectProvider} />
            ))}
          </div>
          <div style={{ height: '1px', background: C.border, margin: '4px 16px' }} />
          <div style={{ padding: '8px 10px 12px' }}>
            <div style={{ fontSize: '11px', color: C.textMuted, padding: '4px 8px', fontWeight: 600 }}>🇨🇳 国内</div>
            {AI_PROVIDERS.filter(p => p.isChinese).map(p => (
              <ProviderItem key={p.id} provider={p} selected={selectedProvider.id === p.id} onSelect={handleSelectProvider} />
            ))}
          </div>
        </aside>

        {/* 右侧配置区 */}
        <div style={{ flex: 1, minWidth: 0 }}>
          {/* 供应商横幅 */}
          <div style={{
            background: C.card, borderRadius: '16px', border: `1px solid ${C.border}`,
            padding: '18px 24px', marginBottom: '20px', boxShadow: '0 2px 8px rgba(0,0,0,0.04)',
            display: 'flex', alignItems: 'center', gap: '14px',
          }}>
            <div style={{
              width: '48px', height: '48px', borderRadius: '12px',
              background: 'linear-gradient(135deg,#EEF2FF,#E0E7FF)',
              display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '24px', flexShrink: 0,
            }}>{selectedProvider.logo}</div>
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: '17px', fontWeight: 700, color: C.text }}>{selectedProvider.name}</div>
              <div style={{ fontSize: '13px', color: C.textSec, marginTop: '2px' }}>{selectedProvider.description}</div>
            </div>
            <button onClick={() => window.open(selectedProvider.docsURL, '_blank', 'noopener,noreferrer')} style={{
              padding: '7px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.bg,
              fontSize: '13px', color: C.primary, cursor: 'pointer', fontWeight: 500,
            }}>查看文档</button>
          </div>

          {/* Tab切换 */}
          <div style={{ display: 'flex', gap: '4px', marginBottom: '16px', background: C.bg, borderRadius: '10px', padding: '4px', border: `1px solid ${C.border}`, width: 'fit-content' }}>
            {(['connection', 'scenes'] as const).map(tab => (
              <button key={tab} onClick={() => setActiveTab(tab)} style={{
                padding: '8px 20px', borderRadius: '8px', border: 'none', cursor: 'pointer',
                fontSize: '14px', fontWeight: activeTab === tab ? 600 : 400,
                color: activeTab === tab ? C.primary : C.textSec,
                background: activeTab === tab ? C.white : 'transparent',
                boxShadow: activeTab === tab ? '0 1px 4px rgba(0,0,0,0.08)' : 'none',
              }}>
                {tab === 'connection' ? '🔌 连接配置' : (
                  <span style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                    ⚙️ 场景配置
                    {modelsQueried && availableModels.length > 0 && (
                      <span style={{ padding: '1px 7px', borderRadius: '10px', fontSize: '11px', background: C.successLight, color: C.success, fontWeight: 600 }}>
                        {availableModels.length}个模型可选
                      </span>
                    )}
                  </span>
                )}
              </button>
            ))}
          </div>

          {/* Tab: 连接配置 */}
          {activeTab === 'connection' && (
            <div style={{ background: C.card, borderRadius: '16px', border: `1px solid ${C.border}`, boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}>
              <div style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}` }}>
                <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>API 连接配置</div>
                <div style={{ fontSize: '13px', color: C.textSec, marginTop: '3px' }}>配置 {selectedProvider.name} 的访问地址和密钥</div>
              </div>
              <div style={{ padding: '24px' }}>
                {/* API Base URL */}
                <div style={{ marginBottom: '16px' }}>
                  <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>API Base URL</label>
                  <input value={form.api_base_url} onChange={e => setForm(p => ({ ...p, api_base_url: e.target.value }))}
                    placeholder={selectedProvider.baseURL}
                    style={{ width: '100%', padding: '10px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: C.white, fontFamily: 'monospace' }}
                    onFocus={e => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = `0 0 0 3px ${C.primaryLight}` }}
                    onBlur={e => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}
                  />
                  <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '3px' }}>默认地址：{selectedProvider.baseURL}</div>
                </div>
                {/* API Key */}
                <div style={{ marginBottom: '16px' }}>
                  <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                    API Key
                    {globalConfig?.api_key_set && <span style={{ fontWeight: 400, color: C.textMuted, marginLeft: '8px', fontSize: '12px' }}>当前：{globalConfig.api_key}（留空不修改）</span>}
                  </label>
                  <div style={{ position: 'relative' }}>
                    <input type={showKey ? 'text' : 'password'} value={form.api_key}
                      onChange={e => setForm(p => ({ ...p, api_key: e.target.value }))}
                      placeholder={globalConfig?.api_key_set ? '留空表示不修改' : `请输入 ${selectedProvider.name} API Key`}
                      style={{ width: '100%', padding: '10px 44px 10px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: C.white, fontFamily: 'monospace' }}
                      onFocus={e => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = `0 0 0 3px ${C.primaryLight}` }}
                      onBlur={e => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}
                    />
                    <button onClick={() => setShowKey(p => !p)} style={{ position: 'absolute', right: '12px', top: '50%', transform: 'translateY(-50%)', background: 'none', border: 'none', cursor: 'pointer', color: C.textMuted, fontSize: '16px' }}>
                      {showKey ? '🙈' : '👁'}
                    </button>
                  </div>
                </div>
                {/* 默认模型 + 温度 + Token */}
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 140px 160px', gap: '14px', marginBottom: '20px' }}>
                  <div>
                    <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                      默认模型{availableModels.length > 0 && <span style={{ fontWeight: 400, color: C.success, marginLeft: '6px', fontSize: '11px' }}>({availableModels.length}个可用)</span>}
                    </label>
                    <ModelSelect value={form.default_model || null} onChange={v => setForm(p => ({ ...p, default_model: v || '' }))} availableModels={availableModels} placeholder="请输入或选择默认模型" />
                  </div>
                  <div>
                    <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>温度</label>
                    <input type="number" step="0.1" min="0" max="2" value={form.temperature} onChange={e => setForm(p => ({ ...p, temperature: e.target.value }))}
                      style={{ width: '100%', padding: '10px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: C.white }}
                      onFocus={e => { e.currentTarget.style.borderColor = C.primary }} onBlur={e => { e.currentTarget.style.borderColor = C.border }} />
                  </div>
                  <div>
                    <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>Max Tokens</label>
                    <input type="number" step="1000" value={form.max_tokens} onChange={e => setForm(p => ({ ...p, max_tokens: e.target.value }))}
                      style={{ width: '100%', padding: '10px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: C.white }}
                      onFocus={e => { e.currentTarget.style.borderColor = C.primary }} onBlur={e => { e.currentTarget.style.borderColor = C.border }} />
                  </div>
                </div>

                {/* 可用模型展示区 */}
                <div style={{
                  borderRadius: '14px', border: `1px solid ${modelsQueried && !modelsError ? 'rgba(16,185,129,0.25)' : C.border}`,
                  background: modelsQueried && !modelsError ? 'rgba(16,185,129,0.03)' : C.bg, marginBottom: '20px', overflow: 'hidden',
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '14px 18px',
                    borderBottom: modelsQueried ? `1px solid ${modelsError ? 'rgba(239,68,68,0.15)' : 'rgba(16,185,129,0.15)'}` : 'none' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <span style={{ fontSize: '15px' }}>📋</span>
                      <span style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>
                        {modelsLoading ? '查询中...' : modelsError ? '查询失败' : modelsQueried ? `当前 Key 可用模型（${availableModels.length} 个）` : '可用模型列表'}
                      </span>
                      {modelsQueried && !modelsError && <span style={{ padding: '2px 8px', borderRadius: '10px', fontSize: '11px', background: C.successLight, color: C.success, fontWeight: 600 }}>已同步到场景配置</span>}
                    </div>
                    <button onClick={handleListModels} disabled={modelsLoading} style={{
                      padding: '7px 16px', borderRadius: '8px', border: 'none',
                      background: modelsLoading ? C.textMuted : 'linear-gradient(135deg,#10B981,#059669)',
                      color: '#fff', fontSize: '13px', fontWeight: 600, cursor: modelsLoading ? 'not-allowed' : 'pointer',
                    }}>{modelsLoading ? '查询中...' : modelsQueried ? '重新查询' : '查询可用模型'}</button>
                  </div>
                  {modelsError && <div style={{ padding: '14px 18px', fontSize: '13px', color: C.danger }}>⚠ {modelsError}</div>}
                  {!modelsQueried && !modelsLoading && (
                    <div style={{ padding: '20px 18px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>
                      点击"查询可用模型"获取当前 API Key 下所有可用模型<br />
                      <span style={{ fontSize: '12px', marginTop: '4px', display: 'block' }}>查询后模型列表将同步到场景配置的下拉菜单和降级模型选择</span>
                    </div>
                  )}
                  {!modelsError && availableModels.length > 0 && (
                    <div style={{ padding: '16px 18px' }}>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
                        {availableModels.map(m => {
                          const isDefault = form.default_model === m
                          return (
                            <button key={m} onClick={() => { setForm(p => ({ ...p, default_model: m })); showToast(`默认模型已设为：${m}`, 'success') }}
                              title="点击设为默认模型"
                              style={{
                                padding: '6px 14px', borderRadius: '8px', border: `1.5px solid ${isDefault ? C.primary : C.border}`,
                                background: isDefault ? C.primaryLight : C.white, color: isDefault ? C.primary : C.text,
                                fontSize: '13px', fontFamily: 'monospace', cursor: 'pointer', fontWeight: isDefault ? 700 : 400,
                                display: 'flex', alignItems: 'center', gap: '6px',
                              }}>{isDefault && <span style={{ fontSize: '10px' }}>✓</span>}{m}</button>
                          )
                        })}
                      </div>
                      <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '10px' }}>点击模型可设为全局默认 · 共 {availableModels.length} 个</div>
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
                      <span style={{ fontWeight: 600, color: testResult.success ? C.success : C.danger }}>{testResult.success ? '连接成功' : '连接失败'}</span>
                      {testResult.latency_ms > 0 && <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: 'rgba(0,0,0,0.06)', color: C.textSec }}>{testResult.latency_ms}ms</span>}
                    </div>
                    <div style={{ fontSize: '13px', color: C.text }}>{testResult.message}</div>
                    {testResult.model && <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>模型：{testResult.model}</div>}
                    <button onClick={() => setTestResult(null)} style={{ marginTop: '10px', padding: '4px 12px', borderRadius: '6px', border: `1px solid ${C.border}`, background: C.white, fontSize: '12px', color: C.textSec, cursor: 'pointer' }}>关闭</button>
                  </div>
                )}

                {/* 操作按钮 */}
                <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
                  <button onClick={handleTest} disabled={testing} style={{
                    padding: '10px 20px', borderRadius: '10px', border: 'none',
                    background: testing ? C.textMuted : 'linear-gradient(135deg,#F59E0B,#D97706)',
                    color: '#fff', fontSize: '14px', fontWeight: 600, cursor: testing ? 'not-allowed' : 'pointer',
                  }}>{testing ? '测试中...' : '🔌 测试连接'}</button>
                  <button onClick={handleSave} disabled={saving} style={{
                    padding: '10px 24px', borderRadius: '10px', border: 'none',
                    background: saving ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`,
                    color: '#fff', fontSize: '14px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer',
                  }}>{saving ? '保存中...' : '💾 保存配置'}</button>
                </div>
              </div>
            </div>
          )}

          {/* Tab: 场景配置（使用拆分后的SceneConfigPanel组件）*/}
          {activeTab === 'scenes' && (
            <SceneConfigPanel
              scenes={scenes}
              editingScene={editingScene}
              sceneForm={sceneForm}
              sceneSaving={sceneSaving}
              availableModels={availableModels}
              modelsQueried={modelsQueried}
              onEditScene={handleEditScene}
              onCancelEdit={() => setEditingScene(null)}
              onSaveScene={handleSaveScene}
              onSceneFormChange={setSceneForm}
            />
          )}
        </div>
      </div>
    </div>
  )
}
