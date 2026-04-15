/**
 * PipelineStepCard.tsx — Pipeline步骤详情卡片 + 步骤面板路由
 *
 *   StepCard        — 可展开的步骤卡片（含断点续跑按钮）
 *   StepPanelRouter — 按步骤名路由到对应面板组件
 */
import { useState } from 'react'
import { RotateCcw, RefreshCw } from 'lucide-react'
import { getStepDetail } from '@/api/pipelines'
import type { StepListItem } from '@/api/pipelines'

// 导入各步骤面板
import DbCheckPanel    from '../steps/DbCheckPanel'
import ScannerPanel    from '../steps/ScannerPanel'
import EvaluatorPanel  from '../steps/EvaluatorPanel'
import MetaPanel       from '../steps/MetaPanel'
import TranslatorPanel from '../steps/TranslatorPanel'
import GeneratorPanel  from '../steps/GeneratorPanel'
import VerifyPanel     from '../steps/VerifyPanel'

// ==================== 常量 ====================

/**
 * 支持断点续跑的步骤集合
 * review是人工审核步骤不支持重跑；verify有独立入口
 */
const RESTARTABLE_STEPS = new Set([
  'dbCheck', 'scanner', 'evaluator', 'meta', 'translator', 'generator',
])

/** 步骤中文名称映射（用于确认弹窗） */
const STEP_CN_MAP: Record<string, string> = {
  dbCheck: '数据检查', scanner: '扫描定位', evaluator: '多轮评估',
  meta: 'Meta综合', translator: '优化翻译', generator: '页面生成',
}

// ==================== 步骤面板路由 ====================

interface StepPanelRouterProps {
  stepName: string
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  data: any
  pipelineId?: string
  pipelineStatus?: string
  stepStatus?: string
  onRefresh?: () => void
}

function StepPanelRouter({ stepName, data, pipelineId, pipelineStatus, stepStatus, onRefresh }: StepPanelRouterProps) {
  if (!data) return <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>

  switch (stepName) {
    case 'dbCheck':    return <DbCheckPanel data={data} />
    case 'scanner':    return <ScannerPanel data={data} />
    case 'evaluator':  return <EvaluatorPanel data={data} />
    case 'meta':       return <MetaPanel data={data} />
    case 'translator':
      return (
        <TranslatorPanel
          data={data}
          pipelineId={pipelineId}
          pipelineStatus={pipelineStatus}
          stepStatus={stepStatus}
          onForceProceed={onRefresh}
        />
      )
    case 'generator':  return <GeneratorPanel data={data} />
    case 'verify':     return <VerifyPanel data={data} pipelineId={pipelineId} />
    default:
      return (
        <pre style={{ fontSize: 12, color: '#3c3c43', overflow: 'auto', maxHeight: 400, margin: 0, whiteSpace: 'pre-wrap' }}>
          {JSON.stringify(data, null, 2)}
        </pre>
      )
  }
}

// ==================== 步骤卡片 ====================

interface StepCardProps {
  pipelineId: string
  step: StepListItem
  pipelineStatus: string
  onRestartFromStep: (stepName: string, stepNameCN: string) => void
  restarting: boolean
  /** v37：高级角色（admin/senior_operator）在更多状态下可见重跑按钮 */
  isHighRole: boolean
}

export function StepCard({
  pipelineId, step, pipelineStatus,
  onRestartFromStep, restarting, isHighRole,
}: StepCardProps) {
  const [expanded, setExpanded]     = useState(false)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [stepData, setStepData]     = useState<any>(null)
  const [loadingData, setLoadingData] = useState(false)

  // 展开/折叠，展开时懒加载步骤数据
  const toggleExpand = async () => {
    if (expanded) { setExpanded(false); return }
    setExpanded(true)
    if (!stepData && step.has_data) {
      setLoadingData(true)
      try {
        const data = await getStepDetail(pipelineId, step.step_name)
        setStepData(data.step_data)
      } catch { /* 静默处理 */ }
      setLoadingData(false)
    }
  }

  /**
   * 断点续跑按钮显示条件（v37改进）：
   * - 步骤在 RESTARTABLE_STEPS 中
   * - 没有正在进行的重跑
   * - Pipeline不在 running/pending 状态
   * - 权限：admin/senior_operator 任何非running状态可见；operator 仅 failed/cancelled
   */
  const showRestartBtn = (() => {
    if (!RESTARTABLE_STEPS.has(step.step_name)) return false
    if (restarting) return false
    if (pipelineStatus === 'running' || pipelineStatus === 'pending') return false
    if (isHighRole) return true
    return pipelineStatus === 'failed' || pipelineStatus === 'cancelled'
  })()

  const statusColors: Record<string, string> = {
    done: '#34c759', running: '#007aff', failed: '#ff3b30',
    skipped: '#aeaeb2', pending: '#c7c7cc',
  }

  return (
    <div style={{
      background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
      border: `1px solid ${step.status === 'failed' ? 'rgba(255,59,48,0.15)' : 'rgba(0,0,0,0.06)'}`,
      borderRadius: 14, overflow: 'hidden',
    }}>
      {/* 卡片头部（点击展开）*/}
      <div
        onClick={toggleExpand}
        style={{
          display: 'flex', alignItems: 'center', gap: 12,
          padding: '14px 18px', cursor: 'pointer',
          borderBottom: expanded ? '1px solid rgba(0,0,0,0.04)' : 'none',
        }}>
        {/* 状态圆点 */}
        <div style={{
          width: 8, height: 8, borderRadius: 4,
          background: statusColors[step.status] || '#c7c7cc', flexShrink: 0,
        }} />

        {/* 步骤名+状态 */}
        <div style={{ flex: 1 }}>
          <span style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>
            {step.step_order}. {step.step_name_cn}
          </span>
          <span style={{ fontSize: 12, color: '#8e8e93', marginLeft: 10 }}>
            {step.status_name}
          </span>
        </div>

        {/* 耗时 */}
        {step.duration_ms > 0 && (
          <span style={{ fontSize: 12, color: '#aeaeb2' }}>
            {step.duration_ms < 1000
              ? step.duration_ms + 'ms'
              : (step.duration_ms / 1000).toFixed(1) + 's'
            }
          </span>
        )}

        {/* Token消耗 */}
        {step.tokens_used > 0 && (
          <span style={{ fontSize: 12, color: '#aeaeb2' }}>
            {step.tokens_used.toLocaleString()} tokens
          </span>
        )}

        {/* 断点续跑按钮 */}
        {showRestartBtn && (
          <button
            onClick={e => {
              e.stopPropagation()
              onRestartFromStep(
                step.step_name,
                STEP_CN_MAP[step.step_name] || step.step_name_cn
              )
            }}
            style={{
              padding: '5px 12px', borderRadius: 8, border: 'none',
              background: 'rgba(255,149,0,0.12)', color: '#ff9500',
              fontSize: 12, fontWeight: 600, cursor: 'pointer',
              display: 'inline-flex', alignItems: 'center', gap: 4, flexShrink: 0,
            }}
            title={`从「${STEP_CN_MAP[step.step_name] || step.step_name_cn}」步骤重新执行`}>
            <RotateCcw size={12} />
            从此步重跑
          </button>
        )}

        {/* 展开箭头 */}
        <span style={{
          fontSize: 12, color: '#c7c7cc',
          transition: 'transform 0.2s',
          transform: expanded ? 'rotate(90deg)' : 'none',
        }}>▶</span>
      </div>

      {/* 展开内容 */}
      {expanded && (
        <div style={{ padding: '14px 18px' }}>
          {/* 错误信息 */}
          {step.status === 'failed' && step.error_message && (
            <div style={{
              padding: '10px 14px', borderRadius: 8, marginBottom: 12,
              background: 'rgba(255,59,48,0.06)', border: '1px solid rgba(255,59,48,0.15)',
              color: '#ff3b30', fontSize: 13, whiteSpace: 'pre-wrap', lineHeight: 1.5,
            }}>
              <div style={{ fontWeight: 600, marginBottom: 4 }}>错误信息：</div>
              {step.error_message}
            </div>
          )}

          {/* 步骤数据 */}
          {loadingData ? (
            <div style={{ color: '#8e8e93', fontSize: 13, display: 'flex', alignItems: 'center', gap: 8 }}>
              <RefreshCw size={13} style={{ animation: 'spin 1s linear infinite' }} />
              加载步骤数据...
            </div>
          ) : stepData ? (
            <StepPanelRouter
              stepName={step.step_name}
              data={stepData}
              pipelineId={pipelineId}
              pipelineStatus={pipelineStatus}
              stepStatus={step.status}
              onRefresh={() => { setStepData(null); toggleExpand() }}
            />
          ) : !step.error_message ? (
            <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>
          ) : null}
        </div>
      )}
    </div>
  )
}
