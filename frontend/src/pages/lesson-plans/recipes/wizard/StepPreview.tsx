/**
 * StepPreview — 配方向导步骤6：预览确认
 *
 * v79 新增：分步向导式配方创建
 *
 * 内容：
 *   - 汇总所有步骤的配置信息，分区展示
 *   - 点击各区域可跳回对应步骤修改
 *   - 底部由向导主页面的"创建配方"按钮触发提交
 *
 * 设计目标：让老师确认一切正确后再提交，减少返工
 */
import {
  STAGE_CODE_EMOJI, STAGE_CODE_NAME,
  PROMPT_MODE_OPTIONS,
} from '../../workshop/components/workshopConstants'
import {
  C, stepCardStyle,
  type WizardFormData,
} from './wizardConstants'

/* ==================== Props 类型 ==================== */
interface StepPreviewProps {
  formData: WizardFormData
  onGoToStep: (step: number) => void
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
export default function StepPreview({ formData, onGoToStep }: StepPreviewProps) {
  const enabledStages = formData.stageFlow.filter(s => s.enabled)
  const modeInfo = PROMPT_MODE_OPTIONS.find(m => m.mode === formData.promptMode)

  // 统计已填写的教师知识字段
  const knowledgeFields = [
    { label: '学情档案', value: formData.studentProfile },
    { label: '教学风格', value: formData.teachingStyle },
    { label: '学校要求', value: formData.schoolRequirements },
    { label: '备课心得', value: formData.customNotes },
    { label: '自定义提示词', value: formData.customPrompt },
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
          确认无误后点击底部「创建配方」完成。
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

      {/* 步骤3：组件 */}
      <PreviewSection
        icon="🧩" title="教学组件" stepIndex={2} onGoToStep={onGoToStep}
        empty={formData.selectedCompIds.size === 0}
      >
        <div style={{ fontSize: '13px', color: formData.selectedCompIds.size > 0 ? C.text : C.textMuted, marginTop: formData.selectedCompIds.size === 0 ? '8px' : '0' }}>
          {formData.selectedCompIds.size > 0
            ? `已选择 ${formData.selectedCompIds.size} 个组件`
            : '未选择，备课时AI会自动匹配推荐组件'}
        </div>
      </PreviewSection>

      {/* 步骤4：教案结构 */}
      <PreviewSection
        icon="📋" title="教案结构" stepIndex={3} onGoToStep={onGoToStep}
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

      {/* 步骤5：备课流程 */}
      <PreviewSection icon="🔧" title="备课流程" stepIndex={4} onGoToStep={onGoToStep}>
        <div style={{ marginBottom: '8px' }}>
          <span style={{ fontSize: '12px', color: C.textMuted }}>备课模式：</span>
          <span style={{ fontSize: '13px', fontWeight: 600, color: C.primary }}>
            {modeInfo ? `${modeInfo.icon} ${modeInfo.label}` : formData.promptMode}
          </span>
        </div>
        <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
          {enabledStages.map((s, i) => (
            <span key={s.stage_code} style={{
              display: 'inline-flex', alignItems: 'center', gap: '4px',
              padding: '4px 10px', borderRadius: '6px', fontSize: '12px',
              background: C.bg, border: `1px solid ${C.border}`, color: C.text,
            }}>
              <span>{STAGE_CODE_EMOJI[s.stage_code] || '📋'}</span>
              <span>{STAGE_CODE_NAME[s.stage_code] || s.stage_code}</span>
              {i < enabledStages.length - 1 && (
                <span style={{ color: C.textMuted, marginLeft: '4px' }}>→</span>
              )}
            </span>
          ))}
        </div>
      </PreviewSection>
    </div>
  )
}
