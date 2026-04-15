/**
 * Translator步骤调试面板
 * P4.5-B: 展示Translator+Reviewer循环详情、最终评分、各轮评分、AI输出
 * v38新增: FAIL时显示原因分析和"确认使用当前方案→启动Generator"按钮
 */
/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState } from 'react'
import {
  kvRow, kvLabel, kvValue, passStyle, failStyle,
  TokenDuration, AIOutputViewer, PromptInfo, SectionTitle, scoreColor,
} from './StepCommon'
import { forceProceed } from '../../../api/pipelines'

/** Translator面板属性 */
interface TranslatorPanelProps {
   
  data: any
  /** Pipeline ID，用于调用forceProceed API */
  pipelineId?: string
  /** Pipeline当前状态，用于判断是否显示强制推进按钮 */
  pipelineStatus?: string
  /** 步骤状态，用于判断是否显示强制推进按钮 */
  stepStatus?: string
  /** 强制推进成功后的回调（刷新页面数据） */
  onForceProceed?: () => void
}

/** Translator步骤调试面板 */
export default function TranslatorPanel({ data, pipelineId, pipelineStatus, stepStatus, onForceProceed }: TranslatorPanelProps) {
  if (!data) return <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>

  return (
    <div>
      {/* 最终结果总览 */}
      <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginBottom: 12, alignItems: 'center' }}>
        <div style={{ textAlign: 'center', minWidth: 90 }}>
          <div style={{ fontSize: 11, color: '#8e8e93', marginBottom: 2 }}>最终评分</div>
          <div style={{ fontSize: 24, fontWeight: 700, color: data.passed ? '#34c759' : '#ff3b30' }}>
            {data.final_score?.toFixed(2) ?? '-'}
          </div>
        </div>
        <div>
          <div style={{ fontSize: 12 }}>
            QUALITY_GATE:{' '}
            <span style={data.final_quality_gate === 'PASS' ? passStyle : failStyle}>
              {data.final_quality_gate || '-'}
            </span>
          </div>
          <div style={{ fontSize: 12, marginTop: 2 }}>
            评级: {data.final_grade || '-'} · 第{data.final_round ?? '-'}轮结束
          </div>
        </div>
        <div style={{ marginLeft: 'auto', textAlign: 'right' }}>
          <div style={{ fontSize: 12, color: '#8e8e93' }}>
            通过: {data.passed ? <span style={passStyle}>是</span> : <span style={failStyle}>否</span>}
          </div>
          <div style={{ fontSize: 12, color: '#aeaeb2', marginTop: 2 }}>
            阈值: {data.threshold ?? 9.0} · 最大轮次: {data.max_loops ?? '-'}
          </div>
        </div>
      </div>

      {/* v38新增：FAIL时显示原因分析和强制推进按钮 */}
      {!data.passed && data.final_trans_output && (
        <FailAnalysisBlock
          data={data}
          pipelineId={pipelineId}
          pipelineStatus={pipelineStatus}
          stepStatus={stepStatus}
          onForceProceed={onForceProceed}
        />
      )}

      {/* 模型/Token信息 */}
      <div style={kvRow}>
        <span style={kvLabel}>模型</span>
        <span style={kvValue}>{data.model_used || '-'}</span>
      </div>
      <TokenDuration tokens={data.total_tokens} durationMs={data.total_latency_ms} />
      <PromptInfo data={data} />

      {/* 各轮循环详情 */}
      {(data.rounds || []).length > 0 && (
        <>
          <SectionTitle title="Translator-Reviewer 循环详情" />
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          {(data.rounds || []).map((r: any) => (  
            <RoundDetail key={r.round} round={r} />
          ))}
        </>
      )}

      {/* 最终Translator输出查看器 */}
      <AIOutputViewer output={data.final_trans_output} label="最终Translator输出（Prompt C）" />

      {/* 最终Reviewer输出查看器 */}
      <AIOutputViewer output={data.final_review_output} label="最终Reviewer输出（Prompt D）" />
    </div>
  )
}

// ==================== v38新增：FAIL原因分析组件 ====================

/** FAIL原因分析块的属性 */
interface FailAnalysisBlockProps {
   
  data: any
  pipelineId?: string
  pipelineStatus?: string
  stepStatus?: string
  onForceProceed?: () => void
}

/**
 * FailAnalysisBlock FAIL原因分析和强制推进按钮
 *
 * 显示内容：
 * 1. FAIL原因摘要（从最后一轮数据分析）
 * 2. 各轮得分趋势
 * 3. "确认使用当前方案→启动Generator"按钮（仅Pipeline状态为failed时显示）
 */
function FailAnalysisBlock({ data, pipelineId, pipelineStatus, stepStatus, onForceProceed }: FailAnalysisBlockProps) {
  const [confirming, setConfirming] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  // 分析FAIL原因
  const lastRound = data.rounds?.[data.rounds.length - 1]
  const finalScore = data.final_score ?? 0
  const threshold = data.threshold ?? 9.0
  const scoreDiff = threshold - finalScore
  const isCloseToPass = scoreDiff > 0 && scoreDiff <= 0.5

  // 找出各轮中最低维度
  const findWeakDimension = () => {
    if (!lastRound) return null
    const dims = [
      { name: 'E1(难度适配)', score: lastRound.e1 },
      { name: 'E2(时间节奏)', score: lastRound.e2 },
      { name: 'E3(互动评估)', score: lastRound.e3 },
      { name: 'E4(课程设计)', score: lastRound.e4 },
    ].filter(d => d.score > 0)
    if (dims.length === 0) return null
    dims.sort((a, b) => a.score - b.score)
    return dims[0]
  }
  const weakDim = findWeakDimension()

  // 判断得分是否在逐轮提升
  const isImproving = () => {
    const rounds = data.rounds || []
    if (rounds.length < 2) return false
    for (let i = 1; i < rounds.length; i++) {
      if ((rounds[i].score ?? 0) < (rounds[i - 1].score ?? 0)) return false
    }
    return true
  }

  // 是否可以显示强制推进按钮
  // 条件：Pipeline状态为failed + Translator步骤为failed + 有Translator输出
  const canForceProceed = pipelineId
    && pipelineStatus === 'failed'
    && stepStatus === 'failed'
    && data.final_trans_output

  // 执行强制推进
  const handleForceProceed = async () => {
    if (!pipelineId) return
    setLoading(true)
    setError('')
    try {
      await forceProceed(pipelineId)
      setSuccess(true)
      setConfirming(false)
      onForceProceed?.()
     
    } catch (e: any) {
      setError(e?.response?.data?.message || e?.message || '操作失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      background: 'rgba(255, 149, 0, 0.06)',
      border: '1px solid rgba(255, 149, 0, 0.2)',
      borderRadius: 8,
      padding: '12px 14px',
      marginBottom: 14,
    }}>
      {/* FAIL原因分析标题 */}
      <div style={{ fontSize: 13, fontWeight: 600, color: '#ff9500', marginBottom: 8 }}>
        💡 FAIL原因分析
      </div>

      {/* 原因说明 */}
      <div style={{ fontSize: 12, color: '#636366', lineHeight: 1.6, marginBottom: 8 }}>
        {isCloseToPass ? (
          <span>
            最终得分 <strong>{finalScore.toFixed(2)}</strong>，距离阈值 <strong>{threshold}</strong> 仅差 <strong>{scoreDiff.toFixed(2)}</strong> 分。
            {isImproving() && ' 各轮得分呈上升趋势，方案质量在持续改善。'}
          </span>
        ) : (
          <span>
            最终得分 <strong>{finalScore.toFixed(2)}</strong>，低于阈值 <strong>{threshold}</strong>。
            QUALITY_GATE判定为FAIL。
          </span>
        )}
        {weakDim && (
          <span>
            {' '}薄弱维度：<strong>{weakDim.name}</strong>（{weakDim.score?.toFixed(1)}分），
            Translator连续{data.rounds?.length ?? 0}轮未能突破该维度。
          </span>
        )}
      </div>

      {/* 评分趋势 */}
      {data.rounds && data.rounds.length > 1 && (
        <div style={{ fontSize: 11, color: '#8e8e93', marginBottom: 10 }}>
          评分趋势：{data.rounds.map((r: any) => r.score?.toFixed(2) ?? '-').join(' → ')}
          {isImproving() && ' 📈'}
        </div>
      )}

      {/* 强制推进按钮区域 */}
      {canForceProceed && !success && (
        <div style={{ marginTop: 4 }}>
          {!confirming ? (
            <button
              onClick={() => setConfirming(true)}
              style={{
                background: '#ff9500',
                color: '#fff',
                border: 'none',
                borderRadius: 6,
                padding: '7px 16px',
                fontSize: 12,
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              ✅ 确认使用当前方案 → 启动Generator
            </button>
          ) : (
            <div style={{
              background: 'rgba(255, 59, 48, 0.06)',
              border: '1px solid rgba(255, 59, 48, 0.2)',
              borderRadius: 6,
              padding: '10px 12px',
            }}>
              <div style={{ fontSize: 12, fontWeight: 600, color: '#ff3b30', marginBottom: 6 }}>
                ⚠️ 确认操作
              </div>
              <div style={{ fontSize: 11, color: '#636366', marginBottom: 8, lineHeight: 1.5 }}>
                将跳过Translator重跑，直接使用最终得分 {finalScore.toFixed(2)} 的方案启动Generator生成页面。
                此操作不可撤销（但后续可从Generator步骤重跑）。
              </div>
              <div style={{ display: 'flex', gap: 8 }}>
                <button
                  onClick={handleForceProceed}
                  disabled={loading}
                  style={{
                    background: loading ? '#aeaeb2' : '#ff3b30',
                    color: '#fff',
                    border: 'none',
                    borderRadius: 4,
                    padding: '5px 14px',
                    fontSize: 12,
                    cursor: loading ? 'not-allowed' : 'pointer',
                  }}
                >
                  {loading ? '执行中...' : '确认启动Generator'}
                </button>
                <button
                  onClick={() => { setConfirming(false); setError('') }}
                  disabled={loading}
                  style={{
                    background: 'transparent',
                    color: '#8e8e93',
                    border: '1px solid #d1d1d6',
                    borderRadius: 4,
                    padding: '5px 14px',
                    fontSize: 12,
                    cursor: 'pointer',
                  }}
                >
                  取消
                </button>
              </div>
              {error && (
                <div style={{ fontSize: 11, color: '#ff3b30', marginTop: 6 }}>{error}</div>
              )}
            </div>
          )}
        </div>
      )}

      {/* 成功提示 */}
      {success && (
        <div style={{
          fontSize: 12,
          color: '#34c759',
          fontWeight: 600,
          marginTop: 4,
        }}>
          ✅ 已启动Generator，请刷新页面查看进度
        </div>
      )}
    </div>
  )
}

// ==================== 单轮详情组件 ====================

/** 单轮Translator-Reviewer详情 */
 
function RoundDetail({ round }: { round: any }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div style={{
      padding: '8px 0', borderTop: '1px solid rgba(0,0,0,0.04)',
    }}>
      {/* 轮次头部（可点击展开AI输出） */}
      <div
        onClick={() => setExpanded(!expanded)}
        style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8 }}
      >
        <span style={{
          transition: 'transform 0.2s', transform: expanded ? 'rotate(90deg)' : 'none',
          fontSize: 10, color: '#c7c7cc',
        }}>▶</span>
        <div style={{ flex: 1 }}>
          <span style={{ fontSize: 13, fontWeight: 600, color: '#1c1c1e' }}>
            第{round.round}轮
          </span>
          <span style={{ marginLeft: 8, fontSize: 12 }}>
            {round.passed ? <span style={passStyle}>通过</span> : <span style={failStyle}>未通过</span>}
          </span>
        </div>
        {/* 分数摘要 */}
        <div style={{ display: 'flex', gap: 10, fontSize: 12, color: '#8e8e93' }}>
          <span>总分: <span style={{ color: scoreColor(round.score), fontWeight: 600 }}>{round.score?.toFixed(2)}</span></span>
          <span>E1: {round.e1?.toFixed(1)}</span>
          <span>E2: {round.e2?.toFixed(1)}</span>
          <span>E3: {round.e3?.toFixed(1)}</span>
          <span>E4: {round.e4?.toFixed(1)}</span>
          <span>GATE: <span style={round.quality_gate === 'PASS' ? passStyle : round.quality_gate === 'FAIL' ? failStyle : { color: '#8e8e93' }}>{round.quality_gate || '-'}</span></span>
        </div>
      </div>

      {/* 展开的详情：Token + AI输出 */}
      {expanded && (
        <div style={{ paddingLeft: 18, marginTop: 8 }}>
          <div style={{ fontSize: 12, color: '#8e8e93', marginBottom: 6 }}>
            Translator: {round.trans_tokens?.toLocaleString() ?? '-'} tokens · {round.trans_latency_ms ? (round.trans_latency_ms / 1000).toFixed(1) + 's' : '-'}
            {round.trans_error && <span style={failStyle}> · 错误: {round.trans_error}</span>}
          </div>
          <div style={{ fontSize: 12, color: '#8e8e93', marginBottom: 8 }}>
            Reviewer: {round.review_tokens?.toLocaleString() ?? '-'} tokens · {round.review_latency_ms ? (round.review_latency_ms / 1000).toFixed(1) + 's' : '-'}
            {round.review_error && <span style={failStyle}> · 错误: {round.review_error}</span>}
          </div>

          {/* Translator输出 */}
          <AIOutputViewer output={round.trans_output} label={`Translator输出（第${round.round}轮）`} maxPreviewLen={300} />

          {/* Reviewer输出 */}
          <AIOutputViewer output={round.review_output} label={`Reviewer输出（第${round.round}轮）`} maxPreviewLen={300} />
        </div>
      )}
    </div>
  )
}

/* eslint-enable @typescript-eslint/no-explicit-any */