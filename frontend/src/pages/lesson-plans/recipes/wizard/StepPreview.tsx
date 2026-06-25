/**
 * StepPreview — 配方向导步骤5：预览确认
 *
 * v79 新增：分步向导式配方创建
 * 动作2（批次A/A.1）：教师知识汇总仅展示学情/学校要求；移除组件预览与备课模式行；
 *   各区块「修改」跳转 stepIndex 随步骤顺移。
 *
 * 【配方搭建一页化 · 批次2B-2】AI 上下文预览并入本步骤（从旧编辑器右栏迁来）：
 *   - 新增 recipeId prop（主页面透传）：编辑态（有 id）可加载 previewRecipeContext，
 *     显示该配方注入 AI 的上下文文本 + 预估 token；新建态（无 id）提示「请先创建配方后查看」。
 *   - 预览按需加载（点「加载预览」/「刷新」才请求），不随每次切步自动拉，省 token、省请求。
 *   - 编辑态下，自定义阶段/教案结构等改动后点「更新配方」落库，再回此步刷新即见最新上下文。
 *
 * 设计目标：让老师确认一切正确后再提交，减少返工
 */
import { useState } from 'react'
import { previewRecipeContext, type RecipeContextPreview } from '@/api/recipes'
import {
  STAGE_CODE_EMOJI, STAGE_CODE_NAME,
} from '../../workshop/components/workshopConstants'
import {
  C, stepCardStyle,
  type WizardFormData,
} from './wizardConstants'

/* ==================== Props 类型 ==================== */
interface StepPreviewProps {
  formData: WizardFormData
  onGoToStep: (step: number) => void
  recipeId?: string   // 批次2B-2：编辑态有值→可加载 AI 上下文预览；新建态 undefined→提示先创建
}

/* ==================== 预览区块组件 ==================== */
function PreviewSection({
  icon, title, stepIndex, onGoToStep, children, empty,
}: {
  icon: string; title: string; stepIndex: number
  onGoToStep: (step: number) => void
  children: React.ReactNode; empty?: boolean
}) {
  return (
    <div style={{
      border: `1px solid ${C.border}`, borderRadius: '10px',
      padding: '16px', marginBottom: '12px',
      background: empty ? 'rgba(156,163,175,0.03)' : '#FAFBFC',
    }}>
      <div style={{
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        marginBottom: empty ? '0' : '10px',
      }}>
        <div style={{
          fontSize: '14px', fontWeight: 600, color: C.text,
        }}>
          {icon} {title}
        </div>
        <button
          onClick={() => onGoToStep(stepIndex)}
          style={{
            fontSize: '12px', color: C.primary, background: 'none',
            border: 'none', cursor: 'pointer',
          }}
        >
          ✏️ 修改
        </button>
      </div>
      {children}
    </div>
  )
}

/* ==================== 组件 ==================== */
export default function StepPreview({ formData, onGoToStep, recipeId }: StepPreviewProps) {
  const isEdit = !!recipeId
  const enabledStages = formData.stageFlow.filter(s => s.enabled)

  // ---- 批次2B-2：AI 上下文预览状态 ----
  const [preview, setPreview] = useState<RecipeContextPreview | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewError, setPreviewError] = useState<string | null>(null)

  const handlePreview = async () => {
    if (!recipeId) return
    setPreviewLoading(true)
    setPreviewError(null)
    try {
      setPreview(await previewRecipeContext(recipeId))
    } catch (e: unknown) {
      setPreviewError(e instanceof Error ? e.message : '预览上下文失败')
    } finally {
      setPreviewLoading(false)
    }
  }

  // 统计已填写的教师知识字段
  const knowledgeFields = [
    { label: '学情档案', value: formData.studentProfile },
    { label: '学校要求', value: formData.schoolRequirements },
  ]
  const filledKnowledge = knowledgeFields.filter(f => f.value.trim().length > 0)

  return (
    <div style={stepCardStyle}>
      {/* 顶部提示 */}
      <div style={{
        padding: '12px 16px', borderRadius: '8px', marginBottom: '20px',
        background: 'rgba(16,185,129,0.06)', border: '1px solid rgba(16,185,129,0.12)',
      }}>
        <div style={{ fontSize: '13px', color: C.success, lineHeight: 1.6 }}>
          ✅ 请确认以下配置。点击各区域右侧的「修改」可跳回对应步骤调整。
          确认无误后点击底部「{isEdit ? '更新配方' : '创建配方'}」完成。
        </div>
      </div>

      {/* 步骤1：基本信息 */}
      <PreviewSection icon="📦" title="基本信息" stepIndex={0} onGoToStep={onGoToStep}>
        <div style={{ display: 'flex', gap: '16px', flexWrap: 'wrap' }}>
          <div>
            <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '2px' }}>配方名称</div>
            <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>
              {formData.name || '(未填写)'}
            </div>
          </div>
          <div>
            <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '2px' }}>学科</div>
            <div style={{ fontSize: '14px', color: C.text }}>{formData.subject}</div>
          </div>
          <div>
            <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '2px' }}>年级</div>
            <div style={{ fontSize: '14px', color: C.text }}>{formData.gradeRange}</div>
          </div>
        </div>
        {formData.description && (
          <div style={{ marginTop: '8px', fontSize: '13px', color: C.textSec }}>
            {formData.description}
          </div>
        )}
      </PreviewSection>

      {/* 步骤2：教师知识 */}
      <PreviewSection
        icon="🧠" title="教师知识" stepIndex={1} onGoToStep={onGoToStep}
        empty={filledKnowledge.length === 0}
      >
        {filledKnowledge.length === 0 ? (
          <div style={{ fontSize: '13px', color: C.textMuted, marginTop: '8px' }}>
            未填写，备课时可随时补充
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            {filledKnowledge.map(f => (
              <div key={f.label} style={{ fontSize: '13px' }}>
                <span style={{ color: C.textSec, fontWeight: 500 }}>{f.label}：</span>
                <span style={{
                  color: C.text, display: 'inline-block', maxWidth: '500px',
                  overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                  verticalAlign: 'bottom',
                }}>
                  {f.value.trim()}
                </span>
              </div>
            ))}
          </div>
        )}
      </PreviewSection>

      {/* 步骤3：教案结构 */}
      <PreviewSection
        icon="📋" title="教案结构" stepIndex={2} onGoToStep={onGoToStep}
        empty={formData.lessonStructure.length === 0}
      >
        {formData.lessonStructure.length === 0 ? (
          <div style={{ fontSize: '13px', color: C.textMuted, marginTop: '8px' }}>
            未定义，AI使用系统默认格式
          </div>
        ) : (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
            {formData.lessonStructure.map((b, i) => (
              <span key={i} style={{
                padding: '4px 10px', borderRadius: '6px', fontSize: '12px',
                background: b.required ? C.primaryLight : C.bg,
                color: b.required ? C.primary : C.textSec,
                border: `1px solid ${b.required ? 'rgba(79,123,232,0.2)' : C.border}`,
              }}>
                {b.name || `板块${i + 1}`}
                {b.required && ' *'}
              </span>
            ))}
          </div>
        )}
      </PreviewSection>

      {/* 步骤4：备课流程 */}
      <PreviewSection icon="🔧" title="备课流程" stepIndex={3} onGoToStep={onGoToStep}>
        <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
          {enabledStages.map((s, i) => (
            <span key={s.stage_code} style={{
              display: 'inline-flex', alignItems: 'center', gap: '4px',
              padding: '4px 10px', borderRadius: '6px', fontSize: '12px',
              background: s.is_custom ? C.primaryLight : C.bg,
              border: `1px solid ${s.is_custom ? 'rgba(79,123,232,0.2)' : C.border}`,
              color: C.text,
            }}>
              <span>{s.is_custom ? '🔧' : (STAGE_CODE_EMOJI[s.stage_code] || '📋')}</span>
              <span>{s.is_custom ? (s.stage_name || s.stage_code) : (STAGE_CODE_NAME[s.stage_code] || s.stage_code)}</span>
              {i < enabledStages.length - 1 && (
                <span style={{ color: C.textMuted, marginLeft: '4px' }}>→</span>
              )}
            </span>
          ))}
        </div>
      </PreviewSection>

      {/* ======== 批次2B-2：AI 上下文预览 ======== */}
      <div style={{
        border: `1px solid ${C.border}`, borderRadius: '10px',
        padding: '16px', marginTop: '4px',
        background: '#FAFBFC',
      }}>
        <div style={{
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          marginBottom: '10px',
        }}>
          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>
            👁️ AI 上下文预览
          </div>
          {isEdit && (
            <button
              onClick={handlePreview}
              disabled={previewLoading}
              style={{
                fontSize: '12px', color: C.primary, background: 'none',
                border: 'none', cursor: previewLoading ? 'default' : 'pointer',
              }}
            >
              {previewLoading ? '加载中...' : (preview ? '🔄 刷新' : '加载预览')}
            </button>
          )}
        </div>

        {/* 新建态：提示先创建 */}
        {!isEdit && (
          <div style={{ fontSize: '13px', color: C.textMuted, padding: '12px 0', lineHeight: 1.6 }}>
            请先创建配方后查看 AI 上下文预览。创建完成后，进入编辑可在此查看该配方注入 AI 的完整上下文与预估 token。
          </div>
        )}

        {/* 编辑态：未加载 */}
        {isEdit && !preview && !previewLoading && !previewError && (
          <div style={{ fontSize: '13px', color: C.textMuted, padding: '8px 0' }}>
            点击右上「加载预览」查看该配方注入 AI 的上下文与预估 token。
          </div>
        )}

        {/* 编辑态：加载出错 */}
        {isEdit && previewError && (
          <div style={{
            fontSize: '12px', color: C.danger, padding: '8px 12px',
            background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.15)',
            borderRadius: '8px',
          }}>
            ⚠️ {previewError}
          </div>
        )}

        {/* 编辑态：预览结果 */}
        {isEdit && preview && (
          <>
            <div style={{
              display: 'flex', alignItems: 'center', gap: '8px',
              padding: '8px 12px', background: 'rgba(16,185,129,0.08)',
              borderRadius: '8px', marginBottom: '12px',
            }}>
              <span style={{ fontSize: '12px' }}>📊</span>
              <span style={{ fontSize: '12px', color: C.success, fontWeight: 600 }}>
                预估 {preview.token_estimate} tokens
              </span>
            </div>
            <div style={{
              maxHeight: '360px', overflowY: 'auto', padding: '12px',
              background: C.card, border: `1px solid ${C.border}`, borderRadius: '8px',
              fontSize: '12px', color: C.textSec, lineHeight: 1.7,
              whiteSpace: 'pre-wrap', wordBreak: 'break-word',
            }}>
              {preview.context_text}
            </div>
          </>
        )}
      </div>
    </div>
  )
}
