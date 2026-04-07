/**
 * StepLessonStructure — 配方向导步骤4：教案结构定义
 *
 * v79 新增：分步向导式配方创建
 *
 * 内容：
 *   - "加载默认模板"一键生成标准教案结构
 *   - 板块编辑器：名称、要求、必填标记、排序
 *   - 教学过程子环节：环节名、时长、目标、输出要求
 *   - 时间轴进度条（课时超限警告）
 *
 * 设计目标：
 *   - 提供默认模板降低从零开始的压力
 *   - 不填时AI使用系统默认格式，可跳过
 */
import { useMemo } from 'react'
import type { LessonStructureBlock, LessonStructureSubSection } from '@/api/recipes'
import {
  C, smallInput, stepCardStyle, DEFAULT_LESSON_STRUCTURE,
  type WizardFormData,
} from './wizardConstants'

/* ==================== Props 类型 ==================== */
interface StepLessonStructureProps {
  formData: WizardFormData
  updateForm: (updates: Partial<WizardFormData>) => void
}

/* ==================== 组件 ==================== */
export default function StepLessonStructure({ formData, updateForm }: StepLessonStructureProps) {
  const ls = formData.lessonStructure

  // 计算教学过程总时长
  const totalDuration = useMemo(() => {
    const b = ls.find(b => b.sub_sections && b.sub_sections.length > 0)
    return b?.sub_sections?.reduce((s, x) => s + (x.duration || 0), 0) || 0
  }, [ls])

  // ---- 板块操作 ----
  const setLS = (next: LessonStructureBlock[]) => updateForm({ lessonStructure: next })

  const loadDefaultStructure = () => setLS(DEFAULT_LESSON_STRUCTURE.map(b => ({ ...b, sub_sections: b.sub_sections?.map(s => ({ ...s })) })))

  const addBlock = () => setLS([...ls, { name: '', required: false, requirement: '', order: ls.length + 1 }])

  const removeBlock = (i: number) => setLS(ls.filter((_, j) => j !== i).map((b, j) => ({ ...b, order: j + 1 })))

  const updateBlock = (i: number, f: keyof LessonStructureBlock, v: unknown) =>
    setLS(ls.map((b, j) => j === i ? { ...b, [f]: v } : b))

  const moveBlock = (i: number, d: -1 | 1) => {
    const t = i + d
    if (t < 0 || t >= ls.length) return
    const n = [...ls]; [n[i], n[t]] = [n[t], n[i]]
    setLS(n.map((b, j) => ({ ...b, order: j + 1 })))
  }

  // ---- 子环节操作 ----
  const addSubSection = (bi: number) =>
    setLS(ls.map((b, i) => i !== bi ? b : {
      ...b,
      sub_sections: [...(b.sub_sections || []), { name: '', duration: 5, goal: '', output_requirement: '' }],
    }))

  const removeSubSection = (bi: number, si: number) =>
    setLS(ls.map((b, i) => i !== bi ? b : {
      ...b,
      sub_sections: (b.sub_sections || []).filter((_, j) => j !== si),
    }))

  const updateSubSection = (bi: number, si: number, f: keyof LessonStructureSubSection, v: unknown) =>
    setLS(ls.map((b, i) => i !== bi ? b : {
      ...b,
      sub_sections: (b.sub_sections || []).map((s, j) => j === si ? { ...s, [f]: v } : s),
    }))

  // 时间轴颜色
  const durationColors = ['#3B82F6', '#10B981', '#F59E0B', '#8B5CF6', '#EC4899', '#14B8A6']

  return (
    <div style={stepCardStyle}>
      {/* 顶部提示 */}
      <div style={{
        padding: '12px 16px', borderRadius: '8px', marginBottom: '20px',
        background: 'rgba(79,123,232,0.06)', border: '1px solid rgba(79,123,232,0.12)',
      }}>
        <div style={{ fontSize: '13px', color: C.primary, lineHeight: 1.6 }}>
          📋 定义你希望教案包含哪些板块（如教学目标、教学过程等）。
          <strong> 不定义时，AI使用系统默认格式。</strong>推荐先「加载默认模板」再微调。
        </div>
      </div>

      {/* 操作按钮 */}
      <div style={{
        display: 'flex', gap: '8px', marginBottom: '16px', justifyContent: 'flex-end',
      }}>
        {ls.length === 0 && (
          <button
            onClick={loadDefaultStructure}
            style={{
              fontSize: '13px', color: C.primary, background: C.primaryLight,
              border: 'none', padding: '8px 16px', borderRadius: '8px',
              cursor: 'pointer', fontWeight: 600,
            }}
          >
            📥 加载默认模板
          </button>
        )}
        <button
          onClick={addBlock}
          style={{
            fontSize: '13px', color: C.success, background: 'rgba(16,185,129,0.08)',
            border: 'none', padding: '8px 16px', borderRadius: '8px',
            cursor: 'pointer', fontWeight: 600,
          }}
        >
          ＋ 添加板块
        </button>
      </div>

      {/* 时长进度条 */}
      {totalDuration > 0 && (
        <div style={{
          marginBottom: '16px', padding: '10px 14px', borderRadius: '8px',
          background: totalDuration > 45 ? 'rgba(239,68,68,0.06)' : 'rgba(16,185,129,0.06)',
          border: `1px solid ${totalDuration > 45 ? 'rgba(239,68,68,0.15)' : 'rgba(16,185,129,0.15)'}`,
        }}>
          <div style={{
            display: 'flex', justifyContent: 'space-between', alignItems: 'center',
            marginBottom: '6px',
          }}>
            <span style={{
              fontSize: '12px', fontWeight: 600,
              color: totalDuration > 45 ? C.danger : C.success,
            }}>
              ⏱ 教学过程合计 {totalDuration} 分钟
            </span>
            {totalDuration > 45 && (
              <span style={{ fontSize: '11px', color: C.danger }}>超出标准课时！</span>
            )}
          </div>
          <div style={{
            display: 'flex', height: '8px', borderRadius: '4px',
            overflow: 'hidden', background: '#E5E7EB',
          }}>
            {ls.find(b => b.sub_sections)?.sub_sections?.map((sub, i) => (
              <div
                key={i}
                style={{
                  width: `${totalDuration > 0 ? (sub.duration / totalDuration) * 100 : 0}%`,
                  background: durationColors[i % durationColors.length],
                  transition: 'width 300ms ease',
                }}
                title={`${sub.name} ${sub.duration}分钟`}
              />
            ))}
          </div>
        </div>
      )}

      {/* 空状态 */}
      {ls.length === 0 && (
        <div style={{
          padding: '40px', textAlign: 'center', color: C.textMuted,
          fontSize: '13px', lineHeight: 1.7,
        }}>
          暂未定义教案结构。点击「加载默认模板」快速开始，或点击「添加板块」自定义。
          <br />
          <span style={{ fontSize: '12px' }}>不定义时，AI使用系统默认格式。</span>
        </div>
      )}

      {/* 板块列表 */}
      {ls.map((block, bIdx) => (
        <div key={bIdx} style={{
          border: `1px solid ${C.border}`, borderRadius: '10px',
          padding: '14px', marginBottom: '10px', background: '#FAFBFC',
        }}>
          {/* 板块头部 */}
          <div style={{
            display: 'flex', gap: '10px', alignItems: 'center', marginBottom: '8px',
          }}>
            {/* 排序 */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
              <button
                onClick={() => moveBlock(bIdx, -1)}
                disabled={bIdx === 0}
                style={{
                  border: 'none', background: 'none', padding: '0',
                  cursor: bIdx === 0 ? 'default' : 'pointer',
                  fontSize: '10px', color: bIdx === 0 ? C.border : C.textMuted,
                }}
              >▲</button>
              <button
                onClick={() => moveBlock(bIdx, 1)}
                disabled={bIdx === ls.length - 1}
                style={{
                  border: 'none', background: 'none', padding: '0',
                  cursor: bIdx === ls.length - 1 ? 'default' : 'pointer',
                  fontSize: '10px', color: bIdx === ls.length - 1 ? C.border : C.textMuted,
                }}
              >▼</button>
            </div>
            {/* 名称 */}
            <input
              value={block.name}
              onChange={e => updateBlock(bIdx, 'name', e.target.value)}
              placeholder="板块名称"
              style={{ ...smallInput, flex: 1, fontWeight: 600 }}
            />
            {/* 必填 */}
            <label style={{
              display: 'flex', alignItems: 'center', gap: '4px',
              fontSize: '12px', color: C.textSec, cursor: 'pointer', whiteSpace: 'nowrap',
            }}>
              <input
                type="checkbox"
                checked={block.required}
                onChange={e => updateBlock(bIdx, 'required', e.target.checked)}
              />
              必填
            </label>
            {/* 删除 */}
            <button
              onClick={() => removeBlock(bIdx)}
              style={{
                border: 'none', background: 'none', cursor: 'pointer',
                fontSize: '14px', color: C.danger, padding: '4px',
              }}
            >✕</button>
          </div>

          {/* 板块要求 */}
          <input
            value={block.requirement}
            onChange={e => updateBlock(bIdx, 'requirement', e.target.value)}
            placeholder="你对这个板块的要求（自然语言描述）"
            style={{ ...smallInput, width: '100%', marginBottom: '8px' }}
          />

          {/* 子环节 */}
          {block.sub_sections && block.sub_sections.length > 0 && (
            <div style={{
              marginTop: '8px', paddingLeft: '12px',
              borderLeft: `3px solid ${C.primary}`,
            }}>
              <div style={{
                fontSize: '12px', fontWeight: 600, color: C.primary, marginBottom: '8px',
              }}>
                教学过程环节安排
              </div>
              {block.sub_sections.map((sub, sIdx) => (
                <div key={sIdx} style={{
                  display: 'flex', gap: '6px', alignItems: 'center',
                  marginBottom: '6px', flexWrap: 'wrap',
                }}>
                  <input
                    value={sub.name}
                    onChange={e => updateSubSection(bIdx, sIdx, 'name', e.target.value)}
                    placeholder="环节名"
                    style={{ ...smallInput, width: '80px' }}
                  />
                  <div style={{ display: 'flex', alignItems: 'center', gap: '3px' }}>
                    <input
                      type="number"
                      value={sub.duration}
                      onChange={e => updateSubSection(bIdx, sIdx, 'duration', parseInt(e.target.value) || 0)}
                      style={{ ...smallInput, width: '50px', textAlign: 'center' }}
                      min={0}
                    />
                    <span style={{ fontSize: '11px', color: C.textMuted }}>分钟</span>
                  </div>
                  <input
                    value={sub.goal}
                    onChange={e => updateSubSection(bIdx, sIdx, 'goal', e.target.value)}
                    placeholder="设计目标"
                    style={{ ...smallInput, flex: 1, minWidth: '100px' }}
                  />
                  <input
                    value={sub.output_requirement}
                    onChange={e => updateSubSection(bIdx, sIdx, 'output_requirement', e.target.value)}
                    placeholder="输出要求"
                    style={{ ...smallInput, flex: 1, minWidth: '100px' }}
                  />
                  <button
                    onClick={() => removeSubSection(bIdx, sIdx)}
                    style={{
                      border: 'none', background: 'none', cursor: 'pointer',
                      fontSize: '12px', color: C.danger, padding: '2px',
                    }}
                  >✕</button>
                </div>
              ))}
              <button
                onClick={() => addSubSection(bIdx)}
                style={{
                  fontSize: '11px', color: C.primary, background: 'none',
                  border: `1px dashed ${C.primary}`, padding: '4px 12px',
                  borderRadius: '4px', cursor: 'pointer', marginTop: '4px',
                }}
              >
                + 添加环节
              </button>
            </div>
          )}

          {/* 教学过程板块但无子环节时的引导 */}
          {(!block.sub_sections || block.sub_sections.length === 0) && block.name.includes('教学过程') && (
            <button
              onClick={() => addSubSection(bIdx)}
              style={{
                fontSize: '11px', color: C.primary, background: 'none',
                border: `1px dashed ${C.primary}`, padding: '4px 12px',
                borderRadius: '4px', cursor: 'pointer',
              }}
            >
              + 添加教学过程子环节
            </button>
          )}
        </div>
      ))}
    </div>
  )
}
