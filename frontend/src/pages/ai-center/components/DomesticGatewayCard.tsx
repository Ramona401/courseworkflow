/**
 * DomesticGatewayCard — 境内文本网关连接配置卡片（批一新增）
 *
 * 挂载位置：AI管理中心 → 连接配置Tab，位于「API连接配置」（境外主网关）卡片之下、TTS卡片之上。
 *
 * 业务背景：
 *   平台 AI 文本调用走「双网关分流」。境外主网关（上方卡片）放行 claude/gemini 等；
 *   未授权学校的境外调用会被**整通道切换**到本卡片配置的「境内网关」（dashscope + qwen-max）。
 *   即：上方=境外通道，本卡=境内降级通道，两套网关物理隔离、密钥不可混用。
 *
 * 功能：
 *   1. 加载当前境内网关配置（base_url / model / api_key脱敏回显）
 *   2. 保存配置：base_url + model + api_key（留空表示不修改）；保存后后端即时失效分流缓存
 *   3. 测试连接：后端用库内三键直连 dashscope 发一句测试请求，回显耗时
 *
 * 后端端点（均admin专属）：
 *   GET  /api/v1/admin/domestic-gateway
 *   PUT  /api/v1/admin/domestic-gateway
 *   POST /api/v1/admin/domestic-gateway/test
 */
import { useState, useEffect, useCallback } from 'react'
import { getDomesticGateway, updateDomesticGateway, testDomesticGateway, getDomesticModels } from '@/api/ai-config'
import type { DomesticGatewayView, DomesticGatewayTestResult } from '@/api/ai-config'
import { C } from './AICenterConstants'

/** 组件Props：复用父页面的Toast通道 */
interface DomesticGatewayCardProps {
  showToast: (message: string, type: 'success' | 'error') => void
}

export default function DomesticGatewayCard({ showToast }: DomesticGatewayCardProps) {
  // ==================== 状态 ====================

  /** 服务端当前配置视图（含脱敏key），null=尚未加载完成 */
  const [view, setView] = useState<DomesticGatewayView | null>(null)
  /** 加载失败信息（403/网络错误时展示，不阻塞整页） */
  const [loadError, setLoadError] = useState<string | null>(null)

  /** 表单：base_url + model + api_key（key留空=不修改） */
  const [form, setForm] = useState({ base_url: '', model: '', api_key: '' })
  /** API Key 明文/密码显隐切换 */
  const [showKey, setShowKey] = useState(false)
  /** 保存中状态 */
  const [saving, setSaving] = useState(false)

  /** 测试中状态 */
  const [testing, setTesting] = useState(false)
  /** 测试结果（null=未测试或已关闭） */
  const [testResult, setTestResult] = useState<DomesticGatewayTestResult | null>(null)

  // 查询可用模型
  const [models, setModels] = useState<string[] | null>(null)
  const [modelsMsg, setModelsMsg] = useState('')
  const [queryingModels, setQueryingModels] = useState(false)

  const queryModels = async () => {
    try {
      setQueryingModels(true)
      setModelsMsg('')
      setModels(null)
      const r = await getDomesticModels()
      setModels(r.models || [])
      if (r.message) setModelsMsg(r.message)
      else if (!r.models || r.models.length === 0) setModelsMsg('网关返回了空模型列表')
    } catch (err: unknown) {
      setModelsMsg(err instanceof Error ? err.message : '查询失败')
    } finally {
      setQueryingModels(false)
    }
  }

  // ==================== 加载当前配置 ====================

  const loadConfig = useCallback(async () => {
    try {
      setLoadError(null)
      const v = await getDomesticGateway()
      setView(v)
      // 回填表单：base_url与model按服务端当前值，key永远留空（留空=不修改）
      setForm({ base_url: v.base_url || '', model: v.model || '', api_key: '' })
    } catch (err: unknown) {
      setLoadError(err instanceof Error ? err.message : '境内网关配置加载失败')
    }
  }, [])

  useEffect(() => { loadConfig() }, [loadConfig])

  // ==================== 事件处理 ====================

  /** 保存配置：后端逐项UPSERT，留空字段不修改，保存成功后后端即时失效分流缓存 */
  const handleSave = async () => {
    try {
      setSaving(true)
      const v = await updateDomesticGateway({
        base_url: form.base_url.trim(),
        model: form.model.trim(),
        api_key: form.api_key.trim(), // 空串后端视为不修改
      })
      setView(v)
      setForm(prev => ({ ...prev, api_key: '' })) // 保存后清空key输入框
      showToast('境内网关配置保存成功（已即时生效）', 'success')
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '境内网关配置保存失败', 'error')
    } finally {
      setSaving(false)
    }
  }

  /** 测试连接：后端直连 dashscope 发测试请求，返回链路结论 */
  const handleTest = async () => {
    try {
      setTesting(true)
      setTestResult(null)
      const r = await testDomesticGateway()
      setTestResult(r)
      showToast(r.success ? '境内网关链路测试成功！' : '境内网关链路测试失败', r.success ? 'success' : 'error')
    } catch (err: unknown) {
      setTestResult({ success: false, message: err instanceof Error ? err.message : '请求失败' })
      showToast('境内网关链路测试失败', 'error')
    } finally {
      setTesting(false)
    }
  }

  // ==================== 公共输入框样式 ====================

  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '10px 14px', borderRadius: '10px',
    border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none',
    boxSizing: 'border-box', background: C.white, fontFamily: 'monospace',
  }

  // ==================== 渲染 ====================

  return (
    <div style={{
      background: C.card, borderRadius: '16px', border: `1px solid ${C.border}`,
      boxShadow: '0 2px 8px rgba(0,0,0,0.04)', marginTop: '20px',
    }}>
      {/* 卡片头部：标题 + 配置状态徽章 */}
      <div style={{
        padding: '18px 24px', borderBottom: `1px solid ${C.border}`,
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
      }}>
        <div>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>🇨🇳 境内网关配置（分流降级通道）</div>
          <div style={{ fontSize: '13px', color: C.textSec, marginTop: '3px' }}>
            未授权学校的境外文本调用会整通道切到这里（dashscope + qwen-max）
          </div>
        </div>
        {view && (
          <div style={{
            padding: '5px 12px', borderRadius: '16px', fontSize: '12px', fontWeight: 600, flexShrink: 0,
            background: view.api_key_set ? C.successLight : C.dangerLight,
            color: view.api_key_set ? C.success : C.danger,
          }}>
            {view.api_key_set ? '✓ Key 已配置' : '⚠ Key 未配置'}
          </div>
        )}
      </div>

      <div style={{ padding: '24px' }}>
        {/* 加载失败提示（不阻塞，可重试） */}
        {loadError && (
          <div style={{
            padding: '12px 16px', borderRadius: '10px', marginBottom: '16px',
            background: C.dangerLight, border: '1px solid rgba(239,68,68,0.25)',
            fontSize: '13px', color: C.danger,
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          }}>
            <span>⚠ {loadError}</span>
            <button onClick={loadConfig} style={{
              padding: '4px 12px', borderRadius: '6px', border: `1px solid ${C.border}`,
              background: C.white, fontSize: '12px', color: C.textSec, cursor: 'pointer',
            }}>重试</button>
          </div>
        )}

        {/* 提示条：与境外网关的关系说明 */}
        <div style={{
          padding: '10px 14px', borderRadius: '10px', marginBottom: '16px',
          background: C.primaryLight, border: `1px solid ${C.primaryBorder}`,
          fontSize: '12px', color: C.textSec, lineHeight: 1.6,
        }}>
          上方「API 连接配置」是<b>境外主网关</b>（claude/gemini 等）；本卡是<b>境内降级网关</b>。
          两套网关密钥物理隔离、<b>不可混用</b>。仅被授权境外的学校走境外，其余一律降级到此境内通道。
        </div>

        {/* 网关地址 base_url */}
        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
            网关地址 Base URL
          </label>
          <input
            value={form.base_url}
            onChange={e => setForm(p => ({ ...p, base_url: e.target.value }))}
            placeholder="https://dashscope.aliyuncs.com/compatible-mode/v1"
            style={inputStyle}
            onFocus={e => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = `0 0 0 3px ${C.primaryLight}` }}
            onBlur={e => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}
          />
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '3px' }}>
            通义千问 dashscope 兼容模式地址，OpenAI 格式 /chat/completions
          </div>
        </div>

        {/* 主力模型 model */}
        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
            主力模型 Model
          </label>
          <input
            value={form.model}
            onChange={e => setForm(p => ({ ...p, model: e.target.value }))}
            placeholder="qwen-max"
            style={inputStyle}
            onFocus={e => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = `0 0 0 3px ${C.primaryLight}` }}
            onBlur={e => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}
          />
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '3px' }}>
            qwen-max 单次输出上限 8192 token，分流时后端会自动夹紧 MaxTokens 避免 400
          </div>
        </div>

        {/* API Key（留空不修改 + 脱敏回显） */}
        <div style={{ marginBottom: '20px' }}>
          <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
            API Key
            {view?.api_key_set && (
              <span style={{ fontWeight: 400, color: C.textMuted, marginLeft: '8px', fontSize: '12px' }}>
                当前：{view.api_key}（留空不修改）
              </span>
            )}
          </label>
          <div style={{ position: 'relative' }}>
            <input
              type={showKey ? 'text' : 'password'}
              value={form.api_key}
              onChange={e => setForm(p => ({ ...p, api_key: e.target.value }))}
              placeholder={view?.api_key_set ? '留空表示不修改' : '请输入 dashscope API Key（sk-...）'}
              style={{ ...inputStyle, padding: '10px 44px 10px 14px' }}
              onFocus={e => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = `0 0 0 3px ${C.primaryLight}` }}
              onBlur={e => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}
            />
            <button onClick={() => setShowKey(p => !p)} style={{
              position: 'absolute', right: '12px', top: '50%', transform: 'translateY(-50%)',
              background: 'none', border: 'none', cursor: 'pointer', color: C.textMuted, fontSize: '16px',
            }}>
              {showKey ? '🙈' : '👁'}
            </button>
          </div>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '3px' }}>
            Key 经 AES 加密存储，仅展示首尾4位 · 与境外网关 Key 不可混用
          </div>
        </div>

        {/* 测试结果展示区 */}
        {testResult && (
          <div style={{
            padding: '16px', borderRadius: '12px', marginBottom: '20px',
            background: testResult.success ? C.successLight : C.dangerLight,
            border: `1px solid ${testResult.success ? 'rgba(16,185,129,0.3)' : 'rgba(239,68,68,0.3)'}`,
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '8px', flexWrap: 'wrap' }}>
              <span>{testResult.success ? '✅' : '❌'}</span>
              <span style={{ fontWeight: 600, color: testResult.success ? C.success : C.danger }}>
                {testResult.success ? '境内通道畅通' : '境内通道失败'}
              </span>
              {typeof testResult.latency_ms === 'number' && testResult.latency_ms > 0 && (
                <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: 'rgba(0,0,0,0.06)', color: C.textSec }}>
                  耗时 {testResult.latency_ms}ms
                </span>
              )}
            </div>
            <div style={{ fontSize: '13px', color: C.text }}>{testResult.message}</div>
            {testResult.model && (
              <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>模型：{testResult.model}</div>
            )}
            <button onClick={() => setTestResult(null)} style={{
              marginTop: '10px', padding: '4px 12px', borderRadius: '6px',
              border: `1px solid ${C.border}`, background: C.white,
              fontSize: '12px', color: C.textSec, cursor: 'pointer',
            }}>关闭</button>
          </div>
        )}

        {/* 查询可用模型（批·查到真实模型名填单价表） */}
        <div style={{ marginBottom: '16px' }}>
          <button onClick={queryModels} disabled={queryingModels} style={{
            padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.primary}`,
            background: C.white, color: C.primary, fontSize: '13px', fontWeight: 600,
            cursor: queryingModels ? 'not-allowed' : 'pointer',
          }}>{queryingModels ? '查询中...' : '🔍 查询可用模型'}</button>

          {modelsMsg && (
            <div style={{ fontSize: '12px', color: C.danger, marginTop: '8px' }}>⚠ {modelsMsg}</div>
          )}

          {models && models.length > 0 && (
            <div style={{ marginTop: '10px', padding: '12px 14px', borderRadius: '10px', background: C.bg, border: `1px solid ${C.border}` }}>
              <div style={{ fontSize: '12px', color: C.textSec, marginBottom: '8px' }}>
                境内网关可用模型（{models.length} 个）— 点模型名复制，填到模型单价表 model_name
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                {models.map(m => (
                  <span key={m}
                    onClick={() => { navigator.clipboard?.writeText(m) }}
                    title="点击复制"
                    style={{ fontSize: '12px', fontFamily: 'monospace', padding: '3px 10px', borderRadius: '6px', background: C.white, border: `1px solid ${C.border}`, color: C.text, cursor: 'pointer' }}>
                    {m}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* 操作按钮：测试连接 + 保存配置 */}
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
          }}>{saving ? '保存中...' : '💾 保存境内网关配置'}</button>
        </div>
      </div>
    </div>
  )
}
