/**
 * CustomSchemeBuilder.tsx — 课件自定义结构搭建器
 *
 * 老师通过点选预置环节模板 + 调页数 + 可选填说明来可视化搭建课件结构。
 * 搭建结果作为结构化文本通过 onChange 回调传出，复用 customPromptHint 管道注入AI。
 * 后端零改动——本组件纯前端UI，产出的文本格式与老师手写描述等效。
 *
 * 交互设计：
 *   - 顶部"快速填充"按钮：一键从小学/初中/高中模板导入全套环节
 *   - 预置环节面板：8种常见环节点击添加
 *   - 已添加环节列表：卡片式，每卡含环节名（可改）+ 页数（按钮组）+ 说明（可选）+ 上移下移删除
 *   - 底部汇总：合计页数 + 结构预览
 */
import { useState, useEffect, useCallback } from 'react'

// ==================== 颜色常量（与workshopConstants的C对齐） ====================
const CC = {
  primary: '#F59E0B',
  primaryBg: 'rgba(245,158,11,0.08)',
  primaryBorder: 'rgba(245,158,11,0.3)',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  white: '#fff',
  danger: '#EF4444',
  success: '#059669',
}

// ==================== 单个环节数据结构 ====================
interface SchemeSection {
  id: string           // 唯一标识（用于key和排序）
  name: string         // 环节名称（可编辑）
  pages: number        // 该环节占几页（1-8）
  note: string         // 可选说明（老师填的补充描述）
  emoji: string        // 环节图标
}

// ==================== 预置环节模板（点击即可添加） ====================
interface SectionTemplate {
  name: string
  emoji: string
  defaultPages: number
  hint: string         // 添加按钮下方的简短说明
}

const SECTION_TEMPLATES: SectionTemplate[] = [
  { name: '封面', emoji: '🎬', defaultPages: 1, hint: '课件标题页' },
  { name: '趣味导入', emoji: '🎯', defaultPages: 1, hint: '故事/问题/游戏引入' },
  { name: '学习目标', emoji: '📋', defaultPages: 1, hint: '本课学习目标' },
  { name: '知识讲解', emoji: '📖', defaultPages: 3, hint: '核心知识图文讲解' },
  { name: '例题解析', emoji: '✍️', defaultPages: 2, hint: '典型例题步骤分解' },
  { name: '互动练习', emoji: '🎮', defaultPages: 2, hint: '拖拽/选择/游戏互动' },
  { name: '创意活动', emoji: '🎨', defaultPages: 1, hint: '动手实践或探究' },
  { name: '课堂总结', emoji: '📝', defaultPages: 1, hint: '知识回顾与作业' },
]

// ==================== 快速模板（一键导入全套环节） ====================
interface QuickTemplate {
  key: string
  label: string
  emoji: string
  sections: Omit<SchemeSection, 'id'>[]
}

const QUICK_TEMPLATES: QuickTemplate[] = [
  {
    key: 'primary', label: '小学模板', emoji: '🎈',
    sections: [
      { name: '封面', pages: 1, note: '', emoji: '🎬' },
      { name: '趣味导入', pages: 1, note: '用故事或游戏引入，激发兴趣', emoji: '🎯' },
      { name: '核心知识', pages: 3, note: '图文为主，文字精简，多用图片', emoji: '📖' },
      { name: '互动练习', pages: 3, note: '拖拽/选择/小游戏，趣味性强', emoji: '🎮' },
      { name: '创意活动', pages: 1, note: '动手实践或小组合作', emoji: '🎨' },
      { name: '趣味总结', pages: 1, note: '轻松回顾，不要大段文字', emoji: '📝' },
    ],
  },
  {
    key: 'middle', label: '初中模板', emoji: '📘',
    sections: [
      { name: '封面', pages: 1, note: '', emoji: '🎬' },
      { name: '学习目标', pages: 1, note: '简洁列出3-5条目标', emoji: '📋' },
      { name: '知识讲解', pages: 4, note: '图文并茂，知识点结构化', emoji: '📖' },
      { name: '例题解析', pages: 3, note: '典型例题，步骤分解', emoji: '✍️' },
      { name: '练习巩固', pages: 3, note: '选择/填空/拖拽混合', emoji: '🎮' },
      { name: '知识小结', pages: 1, note: '思维导图或要点归纳', emoji: '📝' },
      { name: '课后作业', pages: 1, note: '布置作业要求', emoji: '📋' },
    ],
  },
  {
    key: 'high', label: '高中模板', emoji: '🎓',
    sections: [
      { name: '封面', pages: 1, note: '', emoji: '🎬' },
      { name: '学习目标', pages: 1, note: '明确本课目标', emoji: '📋' },
      { name: '知识体系', pages: 5, note: '结构化呈现，知识密度较高', emoji: '📖' },
      { name: '重难点突破', pages: 3, note: '重点难点专项讲解', emoji: '🔥' },
      { name: '例题精讲', pages: 3, note: '含步骤分解的经典例题', emoji: '✍️' },
      { name: '综合练习', pages: 3, note: '综合运用多个知识点', emoji: '🎮' },
      { name: '拓展思考', pages: 1, note: '深层问题与拓展', emoji: '💡' },
      { name: '总结归纳', pages: 1, note: '系统回顾与框架', emoji: '📝' },
    ],
  },
]

// ==================== 生成唯一ID ====================
let _sectionIdCounter = 0
function nextSectionId(): string {
  _sectionIdCounter += 1
  return `sec_${Date.now()}_${_sectionIdCounter}`
}

// ==================== 把环节列表拼成结构化提示词文本 ====================
function buildPromptFromSections(sections: SchemeSection[]): string {
  if (sections.length === 0) return ''
  const totalPages = sections.reduce((sum, s) => sum + s.pages, 0)
  const lines: string[] = []
  lines.push(`【方案结构约束——老师自定义】`)
  lines.push(`- 总页数控制在 ${totalPages} 页`)
  lines.push(`- 严格按照以下环节顺序和页数安排：`)
  sections.forEach((s, i) => {
    let line = `  ${i + 1}. ${s.name}（${s.pages}页）`
    if (s.note.trim()) {
      line += `——${s.note.trim()}`
    }
    lines.push(line)
  })
  lines.push(`- 不要自行增加或删除上述环节，不要改变环节顺序`)
  lines.push(`- 每个环节的页数严格按上述要求分配`)
  return lines.join('\n')
}

// ==================== 组件Props ====================
interface CustomSchemeBuilderProps {
  value: string                    // 当前的 customPromptHint 文本（外部状态）
  onChange: (text: string) => void // 回调：把搭建结果拼成的文本传出
}

// ==================== 主组件 ====================
export default function CustomSchemeBuilder({ value, onChange }: CustomSchemeBuilderProps) {
  const [sections, setSections] = useState<SchemeSection[]>([])
  const [editingNote, setEditingNote] = useState<string | null>(null) // 正在编辑说明的section id

  // 环节变化时，自动拼成文本并回调
  const syncToParent = useCallback((newSections: SchemeSection[]) => {
    setSections(newSections)
    onChange(buildPromptFromSections(newSections))
  }, [onChange])

  // 首次挂载：如果value非空但sections为空，说明是老数据（纯文本），保持不动
  // 如果sections有数据，以sections为准

  // ==================== 操作函数 ====================

  // 从模板添加一个环节
  const addSection = (tpl: SectionTemplate) => {
    const newSection: SchemeSection = {
      id: nextSectionId(),
      name: tpl.name,
      pages: tpl.defaultPages,
      note: '',
      emoji: tpl.emoji,
    }
    syncToParent([...sections, newSection])
  }

  // 快速模板一键填充
  const applyQuickTemplate = (tpl: QuickTemplate) => {
    const newSections: SchemeSection[] = tpl.sections.map(s => ({
      ...s,
      id: nextSectionId(),
    }))
    syncToParent(newSections)
  }

  // 修改页数
  const setPageCount = (id: string, pages: number) => {
    syncToParent(sections.map(s => s.id === id ? { ...s, pages } : s))
  }

  // 修改名称
  const renameName = (id: string, name: string) => {
    syncToParent(sections.map(s => s.id === id ? { ...s, name } : s))
  }

  // 修改说明
  const updateNote = (id: string, note: string) => {
    syncToParent(sections.map(s => s.id === id ? { ...s, note } : s))
  }

  // 删除
  const removeSection = (id: string) => {
    syncToParent(sections.filter(s => s.id !== id))
  }

  // 上移
  const moveUp = (index: number) => {
    if (index <= 0) return
    const arr = [...sections]
    ;[arr[index - 1], arr[index]] = [arr[index], arr[index - 1]]
    syncToParent(arr)
  }

  // 下移
  const moveDown = (index: number) => {
    if (index >= sections.length - 1) return
    const arr = [...sections]
    ;[arr[index], arr[index + 1]] = [arr[index + 1], arr[index]]
    syncToParent(arr)
  }

  // 清空全部
  const clearAll = () => {
    syncToParent([])
  }

  const totalPages = sections.reduce((sum, s) => sum + s.pages, 0)

  return (
    <div style={{ marginBottom: 20, borderRadius: 12, border: `2px solid ${CC.primaryBorder}`, background: CC.primaryBg, overflow: 'hidden' }}>
      {/* 标题栏 */}
      <div style={{ padding: '14px 18px 10px', borderBottom: `1px solid ${CC.primaryBorder}` }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div style={{ fontSize: 15, fontWeight: 700, color: CC.primary }}>✏️ 搭建课件结构</div>
          {sections.length > 0 && (
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <span style={{ fontSize: 14, fontWeight: 700, color: CC.primary }}>
                合计 {totalPages} 页
              </span>
              <button onClick={clearAll} style={{
                fontSize: 12, color: CC.textMuted, background: 'none', border: 'none', cursor: 'pointer',
                textDecoration: 'underline',
              }}>清空</button>
            </div>
          )}
        </div>
        <div style={{ fontSize: 12, color: CC.textSecondary, marginTop: 4 }}>
          点击下方环节快速添加，或一键导入模板后微调
        </div>
      </div>

      {/* 快速模板按钮 */}
      <div style={{ padding: '10px 18px', borderBottom: `1px solid ${CC.primaryBorder}`, display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
        <span style={{ fontSize: 12, color: CC.textMuted, marginRight: 4 }}>快速填充：</span>
        {QUICK_TEMPLATES.map(tpl => (
          <button key={tpl.key} onClick={() => applyQuickTemplate(tpl)}
            style={{
              padding: '5px 14px', borderRadius: 16, fontSize: 12, fontWeight: 600, cursor: 'pointer',
              border: `1px solid ${CC.border}`, background: CC.white, color: CC.textPrimary,
              transition: 'all 150ms',
            }}
            onMouseEnter={e => { (e.target as HTMLElement).style.borderColor = CC.primary; (e.target as HTMLElement).style.color = CC.primary }}
            onMouseLeave={e => { (e.target as HTMLElement).style.borderColor = CC.border; (e.target as HTMLElement).style.color = CC.textPrimary }}
          >
            {tpl.emoji} {tpl.label}
          </button>
        ))}
      </div>

      {/* 预置环节模板面板（点击添加） */}
      <div style={{ padding: '10px 18px', borderBottom: sections.length > 0 ? `1px solid ${CC.primaryBorder}` : 'none', display: 'flex', gap: 6, flexWrap: 'wrap' }}>
        {SECTION_TEMPLATES.map(tpl => (
          <button key={tpl.name} onClick={() => addSection(tpl)}
            title={tpl.hint}
            style={{
              padding: '6px 14px', borderRadius: 20, fontSize: 13, cursor: 'pointer',
              border: `1px dashed ${CC.border}`, background: CC.white, color: CC.textSecondary,
              display: 'flex', alignItems: 'center', gap: 4, transition: 'all 150ms',
            }}
            onMouseEnter={e => { const el = e.currentTarget; el.style.borderColor = CC.primary; el.style.color = CC.primary; el.style.borderStyle = 'solid' }}
            onMouseLeave={e => { const el = e.currentTarget; el.style.borderColor = CC.border; el.style.color = CC.textSecondary; el.style.borderStyle = 'dashed' }}
          >
            <span>{tpl.emoji}</span>
            <span>+ {tpl.name}</span>
          </button>
        ))}
      </div>

      {/* 已添加环节列表 */}
      {sections.length > 0 && (
        <div style={{ padding: '12px 18px' }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {sections.map((sec, idx) => (
              <div key={sec.id} style={{
                display: 'flex', alignItems: 'flex-start', gap: 10, padding: '10px 14px',
                borderRadius: 10, background: CC.white, border: `1px solid ${CC.border}`,
              }}>
                {/* 序号+emoji */}
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2, minWidth: 32, paddingTop: 2 }}>
                  <span style={{
                    width: 24, height: 24, borderRadius: '50%', fontSize: 11, fontWeight: 700,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    background: 'linear-gradient(135deg, #F59E0B, #EF4444)', color: '#fff',
                  }}>{idx + 1}</span>
                  <span style={{ fontSize: 16 }}>{sec.emoji}</span>
                </div>

                {/* 内容区 */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  {/* 环节名（可编辑） */}
                  <input
                    value={sec.name}
                    onChange={e => renameName(sec.id, e.target.value)}
                    style={{
                      fontSize: 14, fontWeight: 600, color: CC.textPrimary, border: 'none',
                      background: 'transparent', outline: 'none', width: '100%', padding: 0,
                    }}
                  />
                  {/* 页数选择（按钮组） */}
                  <div style={{ display: 'flex', gap: 4, marginTop: 6, alignItems: 'center' }}>
                    <span style={{ fontSize: 12, color: CC.textMuted, marginRight: 2 }}>页数：</span>
                    {[1, 2, 3, 4, 5].map(n => (
                      <button key={n} onClick={() => setPageCount(sec.id, n)}
                        style={{
                          width: 28, height: 28, borderRadius: 6, fontSize: 13, fontWeight: 600,
                          border: `1.5px solid ${sec.pages === n ? CC.primary : CC.border}`,
                          background: sec.pages === n ? CC.primary : CC.white,
                          color: sec.pages === n ? '#fff' : CC.textSecondary,
                          cursor: 'pointer', transition: 'all 100ms',
                        }}>{n}</button>
                    ))}
                  </div>
                  {/* 说明（点击展开编辑） */}
                  {editingNote === sec.id ? (
                    <input
                      autoFocus
                      value={sec.note}
                      onChange={e => updateNote(sec.id, e.target.value)}
                      onBlur={() => setEditingNote(null)}
                      onKeyDown={e => { if (e.key === 'Enter') setEditingNote(null) }}
                      placeholder="补充说明（如：用拖拽游戏、多用图片、文字精简...）"
                      style={{
                        width: '100%', marginTop: 6, padding: '4px 8px', borderRadius: 6,
                        border: `1px solid ${CC.primaryBorder}`, fontSize: 12, outline: 'none',
                        color: CC.textPrimary, background: CC.primaryBg,
                      }}
                    />
                  ) : (
                    <div
                      onClick={() => setEditingNote(sec.id)}
                      style={{
                        marginTop: 6, fontSize: 12, cursor: 'pointer', padding: '3px 0',
                        color: sec.note ? CC.textSecondary : CC.textMuted,
                      }}
                    >
                      {sec.note || '+ 添加说明（可选）'}
                    </div>
                  )}
                </div>

                {/* 操作按钮 */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: 2, paddingTop: 2 }}>
                  <button onClick={() => moveUp(idx)} disabled={idx === 0} title="上移"
                    style={{ background: 'none', border: 'none', fontSize: 14, cursor: idx === 0 ? 'default' : 'pointer', opacity: idx === 0 ? 0.25 : 0.6, padding: '1px 4px' }}>▲</button>
                  <button onClick={() => moveDown(idx)} disabled={idx === sections.length - 1} title="下移"
                    style={{ background: 'none', border: 'none', fontSize: 14, cursor: idx === sections.length - 1 ? 'default' : 'pointer', opacity: idx === sections.length - 1 ? 0.25 : 0.6, padding: '1px 4px' }}>▼</button>
                  <button onClick={() => removeSection(sec.id)} title="删除"
                    style={{ background: 'none', border: 'none', fontSize: 14, cursor: 'pointer', color: CC.danger, opacity: 0.6, padding: '1px 4px' }}>✕</button>
                </div>
              </div>
            ))}
          </div>

          {/* 底部汇总 */}
          <div style={{
            marginTop: 10, padding: '8px 14px', borderRadius: 8,
            background: 'rgba(245,158,11,0.12)', fontSize: 13, color: CC.textSecondary,
            display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          }}>
            <span>
              {sections.length} 个环节，合计 <strong style={{ color: CC.primary }}>{totalPages}</strong> 页
            </span>
            <span style={{ fontSize: 12, color: CC.textMuted }}>
              {sections.map(s => `${s.emoji}${s.name}${s.pages}页`).join(' → ')}
            </span>
          </div>
        </div>
      )}

      {/* 空状态提示 */}
      {sections.length === 0 && (
        <div style={{ padding: '24px 18px', textAlign: 'center' }}>
          <div style={{ fontSize: 28, marginBottom: 8 }}>👆</div>
          <div style={{ fontSize: 13, color: CC.textMuted }}>
            点击上方环节添加，或选择「快速填充」一键导入模板
          </div>
        </div>
      )}
    </div>
  )
}
