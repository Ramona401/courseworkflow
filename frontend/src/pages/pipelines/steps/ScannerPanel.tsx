/**
 * Scanner步骤调试面板
 * P4.5-B: 展示课程定位JSON + AI原始输出 + 模型/Token信息
 * 修复：适配实际parsed数据结构（target/ability_targets/grade_standard/course_standard均为对象）
 */
import {
  kvRow, kvLabel, kvValue, passStyle, failStyle,
  TokenDuration, AIOutputViewer, PromptInfo, SectionTitle, JsonViewer,
} from './StepCommon'

/** Scanner面板属性 */
interface ScannerPanelProps {
  data: any
}

/** 安全渲染值：对象转JSON字符串，数组join，其他直接显示 */
function safeRender(val: any): string {
  if (val === null || val === undefined) return '-'
  if (typeof val === 'string') return val || '-'
  if (typeof val === 'number' || typeof val === 'boolean') return String(val)
  if (Array.isArray(val)) {
    return val.map(v => typeof v === 'object' ? JSON.stringify(v) : String(v)).join('；') || '-'
  }
  // 对象：提取关键字段或转JSON
  return JSON.stringify(val, null, 0)
}

/** Scanner步骤调试面板 */
export default function ScannerPanel({ data }: ScannerPanelProps) {
  if (!data) return <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>

  const parsed = data.parsed
  const target = parsed?.target
  const abilityTargets = parsed?.ability_targets
  const gradeStd = parsed?.grade_standard
  const courseStd = parsed?.course_standard

  return (
    <div>
      {/* 基本信息 */}
      <div style={kvRow}>
        <span style={kvLabel}>解析状态</span>
        <span style={kvValue}>
          {data.is_valid
            ? <span style={passStyle}>有效</span>
            : <span style={failStyle}>无效</span>}
        </span>
      </div>
      <div style={kvRow}>
        <span style={kvLabel}>模型</span>
        <span style={kvValue}>{data.model_used || '-'}</span>
      </div>
      <TokenDuration tokens={data.tokens_used} durationMs={data.latency_ms} />
      <PromptInfo data={data} />

      {/* 课程定位信息 — target对象 */}
      {target && typeof target === 'object' && (
        <>
          <SectionTitle title="课程定位（target）" />
          <div style={kvRow}>
            <span style={kvLabel}>课程编号</span>
            <span style={kvValue}>{target.course_id || '-'}</span>
          </div>
          <div style={kvRow}>
            <span style={kvLabel}>课程名称</span>
            <span style={kvValue}>{target.course_name || '-'}</span>
          </div>
          <div style={kvRow}>
            <span style={kvLabel}>年级</span>
            <span style={kvValue}>{target.grade || '-'} ({target.stage || '-'})</span>
          </div>
          <div style={kvRow}>
            <span style={kvLabel}>学期</span>
            <span style={kvValue}>{target.semester || '-'}</span>
          </div>
          <div style={kvRow}>
            <span style={kvLabel}>课程类型</span>
            <span style={kvValue}>{target.course_type || '-'}</span>
          </div>
          <div style={kvRow}>
            <span style={kvLabel}>课时数</span>
            <span style={kvValue}>{target.lesson_count ?? '-'}</span>
          </div>
          <div style={kvRow}>
            <span style={kvLabel}>课时长度</span>
            <span style={kvValue}>{target.lesson_duration_min ? target.lesson_duration_min + '分钟' : '-'}</span>
          </div>
        </>
      )}
      {/* target是字符串的兼容处理 */}
      {target && typeof target === 'string' && (
        <>
          <SectionTitle title="课程定位" />
          <div style={kvRow}>
            <span style={kvLabel}>核心目标</span>
            <span style={kvValue}>{target}</span>
          </div>
        </>
      )}

      {/* 能力目标 — ability_targets对象 */}
      {abilityTargets && (
        <>
          <SectionTitle title="能力目标（ability_targets）" />
          {abilityTargets.targets && Array.isArray(abilityTargets.targets) ? (
            <div style={kvRow}>
              <span style={kvLabel}>目标能力</span>
              <span style={kvValue}>
                {abilityTargets.targets.map((t: any) =>
                  `${t.code || '?'} (L${t.level ?? '?'})`
                ).join('、') || '-'}
              </span>
            </div>
          ) : Array.isArray(abilityTargets) ? (
            <div style={kvRow}>
              <span style={kvLabel}>能力目标</span>
              <span style={kvValue}>{abilityTargets.map((v: any) => safeRender(v)).join('；')}</span>
            </div>
          ) : (
            <div style={kvRow}>
              <span style={kvLabel}>能力目标</span>
              <span style={kvValue}>{safeRender(abilityTargets)}</span>
            </div>
          )}
          {abilityTargets.max_level_code && (
            <div style={kvRow}>
              <span style={kvLabel}>最高等级能力</span>
              <span style={kvValue}>{abilityTargets.max_level_code} (L{abilityTargets.max_level ?? '?'})</span>
            </div>
          )}
          {abilityTargets.target_df_range && (
            <div style={kvRow}>
              <span style={kvLabel}>目标难度范围</span>
              <span style={kvValue}>{abilityTargets.target_df_range.min ?? '?'} ~ {abilityTargets.target_df_range.max ?? '?'}</span>
            </div>
          )}
        </>
      )}

      {/* 年级标准 — grade_standard对象 */}
      {gradeStd && typeof gradeStd === 'object' && (
        <>
          <SectionTitle title="年级标准（grade_standard）" />
          {Object.entries(gradeStd).map(([key, val]) => (
            <div key={key} style={kvRow}>
              <span style={kvLabel}>{key}</span>
              <span style={kvValue}>{safeRender(val)}</span>
            </div>
          ))}
        </>
      )}

      {/* 课程标准 — course_standard对象 */}
      {courseStd && typeof courseStd === 'object' && (
        <>
          <SectionTitle title="课程标准（course_standard）" />
          {Object.entries(courseStd).map(([key, val]) => (
            <div key={key} style={kvRow}>
              <span style={kvLabel}>{key}</span>
              <span style={kvValue}>{safeRender(val)}</span>
            </div>
          ))}
        </>
      )}

      {/* 知识点（如果有） */}
      {parsed?.knowledge_points && (
        <>
          <SectionTitle title="知识点" />
          {parsed.knowledge_points.core && (
            <div style={kvRow}>
              <span style={kvLabel}>核心知识</span>
              <span style={kvValue}>
                {Array.isArray(parsed.knowledge_points.core)
                  ? parsed.knowledge_points.core.join('、')
                  : safeRender(parsed.knowledge_points.core)}
              </span>
            </div>
          )}
          {parsed.knowledge_points.supporting && (
            <div style={kvRow}>
              <span style={kvLabel}>支撑知识</span>
              <span style={kvValue}>
                {Array.isArray(parsed.knowledge_points.supporting)
                  ? parsed.knowledge_points.supporting.join('、')
                  : safeRender(parsed.knowledge_points.supporting)}
              </span>
            </div>
          )}
        </>
      )}

      {/* 螺旋位置（如果有） */}
      {parsed?.spiral_position && (
        <>
          <SectionTitle title="螺旋位置" />
          {parsed.spiral_position.lines && (
            <div style={kvRow}>
              <span style={kvLabel}>所属线索</span>
              <span style={kvValue}>
                {Array.isArray(parsed.spiral_position.lines)
                  ? parsed.spiral_position.lines.join('、')
                  : safeRender(parsed.spiral_position.lines)}
              </span>
            </div>
          )}
          {parsed.spiral_position.position_desc && (
            <div style={kvRow}>
              <span style={kvLabel}>位置描述</span>
              <span style={kvValue}>{parsed.spiral_position.position_desc}</span>
            </div>
          )}
        </>
      )}

      {/* 完整定位JSON查看器 */}
      <JsonViewer data={parsed} label="完整定位JSON" />

      {/* AI原始输出 */}
      <AIOutputViewer output={data.raw_output} label="AI原始输出（Prompt A）" />
    </div>
  )
}
