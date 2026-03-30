/**
 * PipelineDetailBanners.tsx — Pipeline详情页各状态横幅组件
 *
 *   RunningBanner      — 执行中（SSE实时推送 / 轮询回退）
 *   FailedBanner       — 失败/已取消状态提示
 *   MessageBanner      — 验收消息 / 断点续跑消息（通用）
 *   EarlyStopBanner    — 高分早停横幅
 *   VerifiedBanner     — 验收通过横幅
 *   VerifyFailedBanner — 验收未通过横幅
 */
import { Radio, RefreshCw, RotateCcw, ShieldCheck } from 'lucide-react'
import type { PipelineDetailResponse } from '@/api/pipelines'

// ==================== 执行中横幅 ====================

interface RunningBannerProps {
  sseConnected: boolean
  currentStepName: string
}

export function RunningBanner({ sseConnected, currentStepName }: RunningBannerProps) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 10,
      padding: '10px 16px', borderRadius: 10, marginBottom: 16,
      background: sseConnected ? 'rgba(52,199,89,0.06)' : 'rgba(0,122,255,0.06)',
      border: `1px solid ${sseConnected ? 'rgba(52,199,89,0.15)' : 'rgba(0,122,255,0.15)'}`,
    }}>
      {sseConnected
        ? <Radio size={16} color="#34c759" />
        : <RefreshCw size={16} color="#007aff" style={{ animation: 'spin 2s linear infinite' }} />
      }
      <div style={{ fontSize: 13, color: sseConnected ? '#34c759' : '#007aff', fontWeight: 500 }}>
        {sseConnected
          ? 'Pipeline正在执行中，实时推送已连接...'
          : 'Pipeline正在执行中，页面每5秒自动刷新...'
        }
      </div>
      <div style={{ fontSize: 12, color: '#8e8e93' }}>当前步骤: {currentStepName}</div>
      {sseConnected && (
        <span style={{
          marginLeft: 'auto', fontSize: 10, padding: '2px 8px', borderRadius: 10,
          background: 'rgba(52,199,89,0.12)', color: '#34c759', fontWeight: 600,
        }}>SSE</span>
      )}
    </div>
  )
}

// ==================== 失败/取消横幅 ====================

interface FailedBannerProps {
  status: string
  errorMessage?: string
}

export function FailedBanner({ status, errorMessage }: FailedBannerProps) {
  const isFailed = status === 'failed'
  return (
    <div style={{
      display: 'flex', alignItems: 'flex-start', gap: 12,
      padding: '12px 18px', borderRadius: 12, marginBottom: 16,
      background: isFailed
        ? 'linear-gradient(135deg, rgba(255,59,48,0.08), rgba(255,59,48,0.03))'
        : 'linear-gradient(135deg, rgba(142,142,147,0.08), rgba(142,142,147,0.03))',
      border: `1px solid ${isFailed ? 'rgba(255,59,48,0.2)' : 'rgba(142,142,147,0.2)'}`,
    }}>
      <RotateCcw size={18} color={isFailed ? '#ff3b30' : '#8e8e93'} style={{ marginTop: 1, flexShrink: 0 }} />
      <div>
        <div style={{ fontSize: 14, fontWeight: 700, color: isFailed ? '#ff3b30' : '#8e8e93' }}>
          {isFailed ? 'Pipeline执行失败' : 'Pipeline已取消'}
        </div>
        {errorMessage && (
          <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 3 }}>
            错误信息: {errorMessage}
          </div>
        )}
        <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 2 }}>
          展开下方步骤卡片，点击「从此步重跑」可从任意步骤重新执行，无需重建Pipeline。
        </div>
      </div>
    </div>
  )
}

// ==================== 通用消息横幅（验收/重跑）====================

interface MessageBannerProps {
  message: string
  /** 消息类型：verify=验收消息紫色，restart=重跑消息橙色 */
  type: 'verify' | 'restart'
}

export function MessageBanner({ message, type }: MessageBannerProps) {
  const isFailed = message.includes('失败')
  const color  = isFailed ? '#ff3b30' : (type === 'verify' ? '#5856d6' : '#ff9500')
  const bg     = isFailed ? 'rgba(255,59,48,0.08)' : (type === 'verify' ? 'rgba(88,86,214,0.08)' : 'rgba(255,149,0,0.08)')
  const border = isFailed ? 'rgba(255,59,48,0.2)'  : (type === 'verify' ? 'rgba(88,86,214,0.2)' : 'rgba(255,149,0,0.2)')

  return (
    <div style={{
      padding: '10px 16px', borderRadius: 10, marginBottom: 16,
      fontSize: 13, fontWeight: 500,
      display: 'flex', alignItems: 'center', gap: 8,
      background: bg, color, border: `1px solid ${border}`,
    }}>
      {/* 重跑消息且非失败时显示旋转图标 */}
      {type === 'restart' && !isFailed && (
        <RefreshCw size={13} style={{ animation: 'spin 1s linear infinite', flexShrink: 0 }} />
      )}
      {message}
    </div>
  )
}

// ==================== 高分早停横幅 ====================

interface EarlyStopBannerProps {
  detail: PipelineDetailResponse
}

export function EarlyStopBanner({ detail }: EarlyStopBannerProps) {
  // 判断是否触发了高分早停：review步骤有数据，但meta/translator/generator都是pending
  const metaStep  = detail.steps.find(s => s.step_name === 'meta')
  const transStep = detail.steps.find(s => s.step_name === 'translator')
  const genStep   = detail.steps.find(s => s.step_name === 'generator')
  const reviewHasData = detail.steps.some(s => s.step_name === 'review' && s.has_data)

  const isEarlyStop = (
    detail.status === 'review_queue' &&
    reviewHasData &&
    metaStep?.status === 'pending' &&
    transStep?.status === 'pending' &&
    genStep?.status === 'pending'
  )

  if (!isEarlyStop) return null

  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 10,
      padding: '12px 18px', borderRadius: 12, marginBottom: 16,
      background: 'linear-gradient(135deg, rgba(52,199,89,0.1), rgba(52,199,89,0.03))',
      border: '1px solid rgba(52,199,89,0.25)',
    }}>
      <span style={{ fontSize: 20 }}>⚡</span>
      <div>
        <div style={{ fontSize: 14, fontWeight: 700, color: '#34c759' }}>高分早停 — 直接进入审核</div>
        <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 2 }}>
          Evaluator多轮评估均分已达到阈值（{detail.config?.threshold ?? 9.0}分），
          课程质量良好，已跳过Meta优化/翻译/页面生成步骤，直接进入审核队列。
          <span style={{ marginLeft: 6, color: '#34c759', fontWeight: 600 }}>
            可点击「进入审核」查看原始页面。
          </span>
        </div>
      </div>
    </div>
  )
}

// ==================== 验收通过横幅 ====================

export function VerifiedBanner() {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 10,
      padding: '12px 18px', borderRadius: 12, marginBottom: 16,
      background: 'linear-gradient(135deg, rgba(52,199,89,0.1), rgba(52,199,89,0.03))',
      border: '1px solid rgba(52,199,89,0.25)',
    }}>
      <ShieldCheck size={20} color="#34c759" />
      <div>
        <div style={{ fontSize: 14, fontWeight: 700, color: '#34c759' }}>验收通过</div>
        <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 1 }}>
          该Pipeline已通过验收评估，质量达标（≥9.0分）
        </div>
      </div>
    </div>
  )
}

// ==================== 验收未通过横幅 ====================

interface VerifyFailedBannerProps {
  reviewRound: number
}

export function VerifyFailedBanner({ reviewRound }: VerifyFailedBannerProps) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 10,
      padding: '12px 18px', borderRadius: 12, marginBottom: 16,
      background: 'linear-gradient(135deg, rgba(255,149,0,0.1), rgba(255,149,0,0.03))',
      border: '1px solid rgba(255,149,0,0.25)',
    }}>
      <ShieldCheck size={20} color="#ff9500" />
      <div>
        <div style={{ fontSize: 14, fontWeight: 700, color: '#ff9500' }}>验收未通过</div>
        <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 1 }}>
          {reviewRound >= 2
            ? '2审验收仍未达标，需要人工介入处理'
            : '验收评分未达标（<9.0分），系统将自动启动2审流程'
          }
        </div>
      </div>
    </div>
  )
}
