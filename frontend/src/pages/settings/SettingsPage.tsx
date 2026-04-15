/**
 * 系统设置页面（Phase 6 - P6-1b，v81清理）
 *
 * 功能：
 * 1. 健康检查：实时检测后端服务状态
 * 2. 系统信息卡片：版本号、技术栈、部署信息
 * 3. 并发引擎状态（admin可见）：实时查询引擎运行统计
 *
 * v81变更：
 *   - 移除「快捷管理」区域（4个入口已有更好的替代）
 *     → AI配置 / 用户管理 → Header下拉菜单（新版 /ai-center 和 /admin）
 *     → 提示词管理 / 外部数据配置 → 侧边栏已有入口
 *   - 页面更聚焦：健康检查 + 系统信息 + 引擎状态
 *
 * Apple风格内联CSS
 */
import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '@/store/auth'
import client from '@/api/client'
import {
  RefreshCw, Server, Cpu,
  Heart, CheckCircle, XCircle, Loader,
} from 'lucide-react'

// ==================== 类型定义 ====================

/** 健康检查响应 */
interface HealthResponse {
  status: string
  version: string
  time: string
}

/** 引擎状态响应 */
interface EngineStats {
  total_submitted: number
  total_completed: number
  total_failed: number
  current_running: number
  current_ai_active: number
  queue_length: number
  max_workers: number
  max_ai_concurrency: number
  queue_capacity: number
}

// ==================== 信息行组件 ====================
function InfoRow({ label, value, mono }: { label: string; value: string | number; mono?: boolean }) {
  return (
    <div style={{
      display: 'flex', justifyContent: 'space-between', alignItems: 'center',
      padding: '10px 0', borderBottom: '1px solid rgba(0,0,0,0.04)',
    }}>
      <span style={{ fontSize: 13, color: '#86868b', fontWeight: 500 }}>{label}</span>
      <span style={{
        fontSize: 13, color: '#1c1c1e', fontWeight: 600,
        fontFamily: mono ? 'Monaco, Consolas, "Courier New", monospace' : 'inherit',
      }}>{value}</span>
    </div>
  )
}

// ==================== 主页面组件 ====================

export default function SettingsPage() {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  // 健康检查
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [healthLoading, setHealthLoading] = useState(false)
  const [healthError, setHealthError] = useState('')

  // 引擎状态（仅admin）
  const [engine, setEngine] = useState<EngineStats | null>(null)
  const [engineLoading, setEngineLoading] = useState(false)
  const [engineError, setEngineError] = useState('')

  /** 健康检查 */
  const checkHealth = useCallback(async () => {
    setHealthLoading(true)
    setHealthError('')
    try {
      const res = await fetch('/api/v1/health')
      const data = await res.json()
      setHealth(data)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (e: any) {
      setHealthError(e.message || '连接失败')
    }
    setHealthLoading(false)
  }, [])

  /** 获取引擎状态 */
  const loadEngineStats = useCallback(async () => {
    if (!isAdmin) return
    setEngineLoading(true)
    setEngineError('')
    try {
      const res = await client.get('/engine/stats')
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      setEngine((res.data as any).data as EngineStats)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (e: any) {
      setEngineError(e.message || '获取失败')
    }
    setEngineLoading(false)
  }, [isAdmin])

  useEffect(() => {
    // eslint-disable-next-line
    checkHealth()
    loadEngineStats()
  }, [checkHealth, loadEngineStats])

  // ===== 样式 =====
  const card: React.CSSProperties = {
    background: '#fff', borderRadius: 16, padding: '20px 24px',
    border: '1px solid rgba(0,0,0,0.04)', boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
    marginBottom: 20,
  }
  const cardTitle: React.CSSProperties = {
    display: 'flex', alignItems: 'center', gap: 8,
    fontSize: 15, fontWeight: 700, color: '#1c1c1e', marginBottom: 16,
  }
  const btn: React.CSSProperties = {
    padding: '6px 12px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', fontSize: 12, fontWeight: 500, cursor: 'pointer',
    display: 'inline-flex', alignItems: 'center', gap: 4,
  }

  return (
    <div style={{ maxWidth: 800 }}>
      {/* ===== 描述 ===== */}
      <p style={{ fontSize: 14, color: '#86868b', margin: '0 0 20px 0' }}>
        查看系统状态和引擎运行情况
      </p>

      {/* ===== 健康检查 ===== */}
      <div style={card}>
        <div style={{ ...cardTitle, justifyContent: 'space-between' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Heart size={18} color="#ff3b30" />
            <span>服务状态</span>
          </div>
          <button style={btn} onClick={checkHealth} disabled={healthLoading}>
            {healthLoading ? <Loader size={12} style={{ animation: 'spin 1s linear infinite' }} /> : <RefreshCw size={12} />}
            检测
          </button>
        </div>
        {healthError ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: '#ff3b30', fontSize: 13 }}>
            <XCircle size={16} /> 服务异常: {healthError}
          </div>
        ) : health ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: '#34c759', fontSize: 13, fontWeight: 600 }}>
            <CheckCircle size={16} /> 服务正常运行
            <span style={{ color: '#86868b', fontWeight: 400, marginLeft: 8 }}>
              版本 {health.version} · {new Date(health.time).toLocaleString('zh-CN')}
            </span>
          </div>
        ) : (
          <div style={{ color: '#86868b', fontSize: 13 }}>检测中...</div>
        )}
      </div>

      {/* ===== 系统信息 ===== */}
      <div style={card}>
        <div style={cardTitle}>
          <Server size={18} color="#007aff" />
          <span>系统信息</span>
        </div>
        <InfoRow label="系统名称" value="TE-DNA 2.0" />
        <InfoRow label="版本号" value={health?.version || '加载中...'} mono />
        <InfoRow label="技术栈" value="Go 1.22 + React 19 + PostgreSQL 16" />
        <InfoRow label="AI模型" value="anthropic/claude-opus-4.6" />
        <InfoRow label="Pipeline步骤" value="8步（含验收）" />
        <InfoRow label="评估维度" value="E1难度 + E2节奏 + E3互动 + E4设计" />
        <InfoRow label="达标阈值" value="9.0 / 10" />
        <InfoRow label="域名" value="workflow.pkuailab.com" mono />
      </div>

      {/* ===== 并发引擎状态（仅admin） ===== */}
      {isAdmin && (
        <div style={card}>
          <div style={{ ...cardTitle, justifyContent: 'space-between' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Cpu size={18} color="#5856d6" />
              <span>并发引擎</span>
            </div>
            <button style={btn} onClick={loadEngineStats} disabled={engineLoading}>
              {engineLoading ? <Loader size={12} style={{ animation: 'spin 1s linear infinite' }} /> : <RefreshCw size={12} />}
              刷新
            </button>
          </div>
          {engineError ? (
            <div style={{ color: '#ff3b30', fontSize: 13 }}>获取引擎状态失败: {engineError}</div>
          ) : engine ? (
            <>
              {/* 实时指标条 */}
              <div style={{ display: 'flex', gap: 12, marginBottom: 16, flexWrap: 'wrap' }}>
                {[
                  { label: '正在执行', value: engine.current_running, max: engine.max_workers, color: '#007aff' },
                  { label: 'AI活跃', value: engine.current_ai_active, max: engine.max_ai_concurrency, color: '#ff9500' },
                  { label: '队列等待', value: engine.queue_length, max: engine.queue_capacity, color: '#5856d6' },
                ].map(item => (
                  <div key={item.label} style={{
                    flex: 1, minWidth: 120, background: 'rgba(0,0,0,0.02)',
                    borderRadius: 12, padding: '12px 16px',
                  }}>
                    <div style={{ fontSize: 11, color: '#86868b', fontWeight: 600, marginBottom: 6 }}>{item.label}</div>
                    <div style={{ display: 'flex', alignItems: 'baseline', gap: 4 }}>
                      <span style={{ fontSize: 24, fontWeight: 700, color: item.value > 0 ? item.color : '#1c1c1e' }}>
                        {item.value}
                      </span>
                      <span style={{ fontSize: 12, color: '#aeaeb2' }}>/ {item.max}</span>
                    </div>
                    {/* 进度条 */}
                    <div style={{
                      height: 4, background: 'rgba(0,0,0,0.06)', borderRadius: 2,
                      marginTop: 8, overflow: 'hidden',
                    }}>
                      <div style={{
                        width: (item.max > 0 ? (item.value / item.max) * 100 : 0) + '%',
                        height: '100%', background: item.color, borderRadius: 2,
                        transition: 'width 0.3s ease',
                      }} />
                    </div>
                  </div>
                ))}
              </div>
              {/* 累计统计 */}
              <InfoRow label="累计提交任务" value={engine.total_submitted} />
              <InfoRow label="累计完成" value={engine.total_completed} />
              <InfoRow label="累计失败" value={engine.total_failed} />
              <InfoRow label="Worker数量" value={engine.max_workers} />
              <InfoRow label="AI最大并发" value={engine.max_ai_concurrency} />
              <InfoRow label="队列容量" value={engine.queue_capacity} />
            </>
          ) : (
            <div style={{ color: '#86868b', fontSize: 13 }}>加载中...</div>
          )}
        </div>
      )}

      {/* 旋转动画CSS */}
      <style>{`@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`}</style>
    </div>
  )
}
