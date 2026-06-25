/**
 * TTSConfigCard — TTS语音合成连接配置卡片（S-V1.5b新增）
 *
 * 挂载位置：AI管理中心 → 连接配置Tab，位于「API连接配置」卡片之下。
 * 功能：
 *   1. 加载当前TTS配置（provider / APP ID / Access Token脱敏回显 / 内置音色数）
 *   2. 保存配置：provider下拉 + APP ID + Access Token（留空表示不修改）
 *   3. 测试连接：后端用库内配置直连火山合成一句测试音频并即删，回显耗时/时长
 *
 * 后端端点（均admin专属，已上线，本卡片为纯前端补全）：
 *   GET  /api/v1/admin/tts-config
 *   PUT  /api/v1/admin/tts-config
 *   POST /api/v1/admin/tts-config/test
 *
 * 备注：ASR（语音识别）配置卡片上线时直接复制本组件范式。
 */
import { useState, useEffect, useCallback } from 'react'
import { getTTSConfig, updateTTSConfig, testTTSConnection } from '@/api/ai-config'
import type { TTSConfigView, TestTTSResult } from '@/api/ai-config'
import { C } from './AICenterConstants'

/** provider 选项字典（值与后端 ai.TTSProviderVolcanoV3 / TTSProviderOpenAI 常量严格一致） */
const TTS_PROVIDERS = [
  { value: 'volcano_v3', label: '火山豆包语音 v3 直连（默认，推荐）' },
  { value: 'volcano_openai', label: 'OpenAI兼容通道（备用）' },
]

/** 组件Props：复用父页面的Toast通道 */
interface TTSConfigCardProps {
  showToast: (message: string, type: 'success' | 'error') => void
}

export default function TTSConfigCard({ showToast }: TTSConfigCardProps) {
  // ==================== 状态 ====================

  /** 服务端当前配置视图（含脱敏token），null=尚未加载完成 */
  const [view, setView] = useState<TTSConfigView | null>(null)
  /** 加载失败信息（403/网络错误时展示，不阻塞整页） */
  const [loadError, setLoadError] = useState<string | null>(null)

  /** 表单：provider下拉 + APP ID + Access Token（token留空=不修改） */
  const [form, setForm] = useState({ provider: 'volcano_v3', app_id: '', access_token: '' })
  /** Access Token明文/密码显隐切换 */
  const [showToken, setShowToken] = useState(false)
  /** 保存中状态 */
  const [saving, setSaving] = useState(false)

  /** 测试中状态（合成一句音频通常2-8秒） */
  const [testing, setTesting] = useState(false)
  /** 测试结果（null=未测试或已关闭） */
  const [testResult, setTestResult] = useState<TestTTSResult | null>(null)

  // ==================== 加载当前配置 ====================

  const loadConfig = useCallback(async () => {
    try {
      setLoadError(null)
      const v = await getTTSConfig()
      setView(v)
      // 回填表单：provider与APP ID按服务端当前值，token永远留空（留空=不修改）
      setForm({ provider: v.provider || 'volcano_v3', app_id: v.app_id || '', access_token: '' })
    } catch (err: unknown) {
      setLoadError(err instanceof Error ? err.message : 'TTS配置加载失败')
    }
  }, [])

  useEffect(() => { loadConfig() }, [loadConfig])

  // ==================== 事件处理 ====================

  /** 保存配置：后端逐项UPSERT，留空字段不修改 */
  const handleSave = async () => {
    try {
      setSaving(true)
      const v = await updateTTSConfig({
        provider: form.provider,
        app_id: form.app_id.trim(),
        access_token: form.access_token.trim(), // 空串后端视为不修改
      })
      setView(v)
      setForm(prev => ({ ...prev, access_token: '' })) // 保存后清空token输入框
      showToast('TTS配置保存成功', 'success')
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : 'TTS配置保存失败', 'error')
    } finally {
      setSaving(false)
    }
  }

  /** 测试连接：后端直连合成测试音频并即删，返回链路结论 */
  const handleTest = async () => {
    try {
      setTesting(true)
      setTestResult(null)
      const r = await testTTSConnection()
      setTestResult(r)
      showToast(r.success ? 'TTS链路测试成功！' : 'TTS链路测试失败', r.success ? 'success' : 'error')
    } catch (err: unknown) {
      setTestResult({ success: false, message: err instanceof Error ? err.message : '请求失败' })
      showToast('TTS链路测试失败', 'error')
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
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>🎙 TTS 语音合成配置</div>
          <div style={{ fontSize: '13px', color: C.textSec, marginTop: '3px' }}>
            字幕轨批量配音使用的火山豆包语音通道
            {view && view.voices_total > 0 && `（内置 ${view.voices_total} 个音色）`}
          </div>
        </div>
        {view && (
          <div style={{
            padding: '5px 12px', borderRadius: '16px', fontSize: '12px', fontWeight: 600, flexShrink: 0,
            background: view.access_token_set ? C.successLight : C.dangerLight,
            color: view.access_token_set ? C.success : C.danger,
          }}>
            {view.access_token_set ? '✓ Token 已配置' : '⚠ Token 未配置'}
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

        {/* Provider 下拉 */}
        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
            合成通道 Provider
          </label>
          <select
            value={form.provider}
            onChange={e => setForm(p => ({ ...p, provider: e.target.value }))}
            style={{
              width: '100%', padding: '10px 14px', borderRadius: '10px',
              border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none',
              background: C.white, cursor: 'pointer', color: C.text, boxSizing: 'border-box',
            }}
            onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
            onBlur={e => { e.currentTarget.style.borderColor = C.border }}
          >
            {TTS_PROVIDERS.map(p => (
              <option key={p.value} value={p.value}>{p.label}</option>
            ))}
          </select>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '3px' }}>
            v3直连通道按音色码自动推导resource，无需配置模型名
          </div>
        </div>

        {/* APP ID */}
        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
            APP ID
          </label>
          <input
            value={form.app_id}
            onChange={e => setForm(p => ({ ...p, app_id: e.target.value }))}
            placeholder="火山「豆包语音」控制台的应用 APP ID"
            style={inputStyle}
            onFocus={e => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = `0 0 0 3px ${C.primaryLight}` }}
            onBlur={e => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}
          />
        </div>

        {/* Access Token（留空不修改 + 脱敏回显） */}
        <div style={{ marginBottom: '20px' }}>
          <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
            Access Token
            {view?.access_token_set && (
              <span style={{ fontWeight: 400, color: C.textMuted, marginLeft: '8px', fontSize: '12px' }}>
                当前：{view.access_token}（留空不修改）
              </span>
            )}
          </label>
          <div style={{ position: 'relative' }}>
            <input
              type={showToken ? 'text' : 'password'}
              value={form.access_token}
              onChange={e => setForm(p => ({ ...p, access_token: e.target.value }))}
              placeholder={view?.access_token_set ? '留空表示不修改' : '请输入火山豆包语音 Access Token'}
              style={{ ...inputStyle, padding: '10px 44px 10px 14px' }}
              onFocus={e => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = `0 0 0 3px ${C.primaryLight}` }}
              onBlur={e => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}
            />
            <button onClick={() => setShowToken(p => !p)} style={{
              position: 'absolute', right: '12px', top: '50%', transform: 'translateY(-50%)',
              background: 'none', border: 'none', cursor: 'pointer', color: C.textMuted, fontSize: '16px',
            }}>
              {showToken ? '🙈' : '👁'}
            </button>
          </div>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '3px' }}>
            Token 经 AES 加密存储，仅展示首尾4位
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
                {testResult.success ? 'TTS 链路畅通' : 'TTS 链路失败'}
              </span>
              {typeof testResult.latency_ms === 'number' && testResult.latency_ms > 0 && (
                <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: 'rgba(0,0,0,0.06)', color: C.textSec }}>
                  耗时 {testResult.latency_ms}ms
                </span>
              )}
              {typeof testResult.duration === 'number' && testResult.duration > 0 && (
                <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: 'rgba(0,0,0,0.06)', color: C.textSec }}>
                  音频 {testResult.duration.toFixed(1)}s
                </span>
              )}
            </div>
            <div style={{ fontSize: '13px', color: C.text }}>{testResult.message}</div>
            {testResult.model && (
              <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>resource：{testResult.model}</div>
            )}
            <button onClick={() => setTestResult(null)} style={{
              marginTop: '10px', padding: '4px 12px', borderRadius: '6px',
              border: `1px solid ${C.border}`, background: C.white,
              fontSize: '12px', color: C.textSec, cursor: 'pointer',
            }}>关闭</button>
          </div>
        )}

        {/* 操作按钮：测试连接 + 保存配置 */}
        <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
          <button onClick={handleTest} disabled={testing} style={{
            padding: '10px 20px', borderRadius: '10px', border: 'none',
            background: testing ? C.textMuted : 'linear-gradient(135deg,#F59E0B,#D97706)',
            color: '#fff', fontSize: '14px', fontWeight: 600, cursor: testing ? 'not-allowed' : 'pointer',
          }}>{testing ? '合成测试中...' : '🔊 测试连接'}</button>
          <button onClick={handleSave} disabled={saving} style={{
            padding: '10px 24px', borderRadius: '10px', border: 'none',
            background: saving ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`,
            color: '#fff', fontSize: '14px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer',
          }}>{saving ? '保存中...' : '💾 保存TTS配置'}</button>
        </div>
      </div>
    </div>
  )
}
