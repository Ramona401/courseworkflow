/**
 * GatewayNamingCard — 双网关展示名配置卡片（批三-1新增）
 *
 * 挂载位置：AI管理中心 → 连接配置Tab 顶部（境外主网关卡片之前）。
 *
 * 业务背景：
 *   平台 AI 文本调用走「双网关分流」——境外主网关（claude/gemini）+ 境内网关（qwen）。
 *   为避免对外（尤其老师侧）直接暴露"境外/claude/qwen"等字眼，给两个网关各起一个
 *   业务可读展示名（如"星云国际通道"/"星云境内通道"）。
 *
 * 功能：
 *   1. 加载当前两网关展示名（getGatewayNaming，未配置回显占位提示）
 *   2. 保存（updateGatewayNaming，两字段留空=不修改对应项）
 *
 * 后端端点（admin专属）：GET/PUT /api/v1/admin/gateway-naming
 *
 * 说明：本卡片仅做 admin 侧存/取。老师侧读取这两个名字（不暴露真实模型）在批三-3统一接。
 */
import { useState, useEffect, useCallback } from 'react'
import { getGatewayNaming, updateGatewayNaming } from '@/api/ai-config'
import { C } from './AICenterConstants'

interface GatewayNamingCardProps {
  showToast: (message: string, type: 'success' | 'error') => void
}

export default function GatewayNamingCard({ showToast }: GatewayNamingCardProps) {
  /** 表单：境外名 + 境内名 */
  const [overseasLabel, setOverseasLabel] = useState('')
  const [domesticLabel, setDomesticLabel] = useState('')
  const [loaded, setLoaded] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  // ==================== 加载当前展示名 ====================
  const loadNaming = useCallback(async () => {
    try {
      setLoadError(null)
      const v = await getGatewayNaming()
      setOverseasLabel(v.overseas_label || '')
      setDomesticLabel(v.domestic_label || '')
      setLoaded(true)
    } catch (err: unknown) {
      setLoadError(err instanceof Error ? err.message : '网关展示名加载失败')
    }
  }, [])

  useEffect(() => { loadNaming() }, [loadNaming])

  // ==================== 保存 ====================
  const handleSave = async () => {
    try {
      setSaving(true)
      // 两字段都传当前值（空串后端视为不修改；若要清空需另设逻辑，此处保持"留空不改"语义）
      const v = await updateGatewayNaming({
        overseas_label: overseasLabel.trim(),
        domestic_label: domesticLabel.trim(),
      })
      setOverseasLabel(v.overseas_label || '')
      setDomesticLabel(v.domestic_label || '')
      showToast('网关展示名保存成功', 'success')
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '网关展示名保存失败', 'error')
    } finally {
      setSaving(false)
    }
  }

  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '10px 14px', borderRadius: '10px',
    border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none',
    boxSizing: 'border-box', background: C.white,
  }

  return (
    <div style={{
      background: C.card, borderRadius: '16px', border: `1px solid ${C.border}`,
      boxShadow: '0 2px 8px rgba(0,0,0,0.04)', marginBottom: '20px',
    }}>
      {/* 头部 */}
      <div style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}` }}>
        <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>🏷️ 网关命名（对外展示名）</div>
        <div style={{ fontSize: '13px', color: C.textSec, marginTop: '3px' }}>
          给境内/境外两网关起业务可读名，避免对外暴露厂商/模型字眼（将来老师侧也读此名）
        </div>
      </div>

      <div style={{ padding: '24px' }}>
        {/* 加载失败提示 */}
        {loadError && (
          <div style={{
            padding: '12px 16px', borderRadius: '10px', marginBottom: '16px',
            background: C.dangerLight, border: '1px solid rgba(239,68,68,0.25)',
            fontSize: '13px', color: C.danger,
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          }}>
            <span>⚠ {loadError}</span>
            <button onClick={loadNaming} style={{
              padding: '4px 12px', borderRadius: '6px', border: `1px solid ${C.border}`,
              background: C.white, fontSize: '12px', color: C.textSec, cursor: 'pointer',
            }}>重试</button>
          </div>
        )}

        {/* 说明条 */}
        <div style={{
          padding: '10px 14px', borderRadius: '10px', marginBottom: '18px',
          background: C.primaryLight, border: `1px solid ${C.primaryBorder}`,
          fontSize: '12px', color: C.textSec, lineHeight: 1.6,
        }}>
          这两个名字是<b>对外展示标识</b>，不影响实际调用。例如把境外网关命名为"星云国际通道"、境内命名为"星云境内通道"。留空则前端用默认名兜底。
        </div>

        {/* 境外网关展示名 */}
        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
            境外网关展示名
          </label>
          <input
            value={overseasLabel}
            onChange={e => setOverseasLabel(e.target.value)}
            placeholder="例如：星云国际通道"
            style={inputStyle}
            onFocus={e => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = `0 0 0 3px ${C.primaryLight}` }}
            onBlur={e => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}
          />
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '3px' }}>
            对应上方「API 连接配置」境外主网关（claude/gemini 等）
          </div>
        </div>

        {/* 境内网关展示名 */}
        <div style={{ marginBottom: '20px' }}>
          <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
            境内网关展示名
          </label>
          <input
            value={domesticLabel}
            onChange={e => setDomesticLabel(e.target.value)}
            placeholder="例如：星云境内通道"
            style={inputStyle}
            onFocus={e => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = `0 0 0 3px ${C.primaryLight}` }}
            onBlur={e => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}
          />
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '3px' }}>
            对应「境内网关配置」降级通道（qwen-max）
          </div>
        </div>

        {/* 保存按钮 */}
        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
          <button onClick={handleSave} disabled={saving || !loaded} style={{
            padding: '10px 24px', borderRadius: '10px', border: 'none',
            background: (saving || !loaded) ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`,
            color: '#fff', fontSize: '14px', fontWeight: 600,
            cursor: (saving || !loaded) ? 'not-allowed' : 'pointer',
          }}>{saving ? '保存中...' : '💾 保存网关命名'}</button>
        </div>
      </div>
    </div>
  )
}
