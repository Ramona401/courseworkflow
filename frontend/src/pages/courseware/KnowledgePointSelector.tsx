/**
 * KnowledgePointSelector — 课标知识点选择器（平台级可复用组件）
 *
 * 用途：传入学科 + 年级（文字，如"三年级"），自动拉取该年级课标知识点，
 *       按领域分组渲染为可勾选清单；选中的 kp_code 数组通过 onChange 回传父组件。
 *
 * 复用场景：
 *   - 课件工坊「从主题创建」：勾选知识点 → 难度自动适配
 *   - 备课工坊「教案撰写」（将来）：同样复用本组件选基础数据
 *
 * 设计：自包含——内部完成"年级文字→数字"转换、拉取、分组、加载/空态处理；
 *       父组件只需给 subject/grade 文字 + 收 onChange(codes)。
 *
 * 本轮修复（迭代回顾）：
 *   - 修复脏数据入口：subject/年级变化时自动清空已勾选，杜绝"语文课件绑定数学知识点"。
 *   - 增强年级解析：支持纯数字、高中(高一~高三/10~12年级)、"小学/初中X年级"前缀。
 */
import { useState, useEffect, useRef } from 'react'
import { getCurriculumKnowledgePoints } from '@/api/coursewares'
import type { CurriculumKP } from '@/api/coursewares'
import { CW_DEPTH_LEVEL_CONFIG } from '@/api/coursewares'

// 年级文字 → 数字（义务教育1-9年级 + 高中10-12年级）。
// 识别"三年级/初二/初中二年级/七年级/高一/高二/十年级/3"等常见写法；无法识别返回 0。
// 注意：高中(10-12)目前知识库可能暂无数据，解析对了选择器会显示"暂无数据"而非"填写有效年级"，体验更准确。
export function parseGradeNum(gradeText: string): number {
  if (!gradeText) return 0
  // 去掉"小学/初中/高中"等前缀干扰词，避免"小学三年级"里的"小"等被误判
  const t = gradeText.trim()

  // 1. 高中优先识别（高一/高二/高三 → 10/11/12）
  if (/高一|高中一|高1/.test(t)) return 10
  if (/高二|高中二|高2/.test(t)) return 11
  if (/高三|高中三|高3/.test(t)) return 12

  // 2. 初中（初一/初二/初三 → 7/8/9）
  if (/初一|初中一|初1/.test(t)) return 7
  if (/初二|初中二|初2/.test(t)) return 8
  if (/初三|初中三|初3/.test(t)) return 9

  // 3. 阿拉伯数字 + "年级"：如"3年级""10年级""12年级"
  const arabGrade = t.match(/(1[0-2]|[1-9])\s*年级/)
  if (arabGrade) return parseInt(arabGrade[1], 10)

  // 4. 中文数字年级：一~九年级 + 十/十一/十二年级
  const cnMap: Record<string, number> = {
    一: 1, 二: 2, 三: 3, 四: 4, 五: 5, 六: 6, 七: 7, 八: 8, 九: 9,
    十: 10, 十一: 11, 十二: 12,
  }
  // 先匹配两字（十一/十二），再匹配单字，避免"十一"被"十"截断
  const cn2 = t.match(/(十一|十二)年级/)
  if (cn2) return cnMap[cn2[1]]
  const cn1 = t.match(/([一二三四五六七八九十])年级/)
  if (cn1) return cnMap[cn1[1]]

  // 5. 纯数字（无"年级"二字）：如用户只填"3""10"
  const pureNum = t.match(/^(1[0-2]|[1-9])$/)
  if (pureNum) return parseInt(pureNum[1], 10)

  return 0
}

const COLORS = {
  primary: '#7C3AED', border: '#E5E7EB',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
  domainBg: '#F9FAFB',
}

interface Props {
  subject: string          // 学科（文字）
  grade: string            // 年级（文字，如"三年级"）
  selectedCodes: string[]  // 当前已选 kp_code 数组（受控）
  onChange: (codes: string[]) => void  // 选择变化回调
}

export default function KnowledgePointSelector({ subject, grade, selectedCodes, onChange }: Props) {
  const [kps, setKps] = useState<CurriculumKP[]>([])
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)  // 是否已尝试过加载（区分"未查"与"查了但为空"）
  const [err, setErr] = useState('')

  const gradeNum = parseGradeNum(grade)

  // 脏数据防护：记录上一次的 subject+gradeNum，仅在"真正切换"时清空已勾选。
  // 用 ref 避免把 onChange 放进 useEffect 依赖（onChange 每次渲染都是新函数会导致误触发）。
  const prevKeyRef = useRef<string>(`${subject}|${gradeNum}`)

  // 学科或年级变化时：① 清空已勾选（杜绝跨学科年级脏数据）② 重新拉取知识点
  useEffect(() => {
    const curKey = `${subject}|${gradeNum}`
    // 仅当 key 真正变化且确实已有勾选时才清空，避免首次挂载/无谓清空
    if (prevKeyRef.current !== curKey) {
      prevKeyRef.current = curKey
      if (selectedCodes.length > 0) {
        onChange([])
      }
    }

    if (!subject || gradeNum <= 0) {
      setKps([]); setLoaded(false); setErr('')
      return
    }
    let cancelled = false
    setLoading(true); setErr('')
    getCurriculumKnowledgePoints(subject, gradeNum)
      .then(resp => {
        if (cancelled) return
        setKps(resp.knowledge_points || [])
        setLoaded(true)
      })
      .catch(() => {
        if (cancelled) return
        setErr('知识点加载失败')
        setKps([]); setLoaded(true)
      })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
    // 依赖只放 subject 和 gradeNum；selectedCodes/onChange 故意不放，避免循环触发
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [subject, gradeNum])

  // 未选学科或年级无法识别：提示但不报错
  if (!subject || gradeNum <= 0) {
    return (
      <div style={{ fontSize: '12px', color: COLORS.textMuted, padding: '8px 0' }}>
        选择学科并填写有效年级（如"三年级""初二""高一"）后，将显示该年级课标知识点供勾选
      </div>
    )
  }

  if (loading) {
    return <div style={{ fontSize: '13px', color: COLORS.textMuted, padding: '12px 0' }}>加载知识点...</div>
  }

  if (err) {
    return <div style={{ fontSize: '13px', color: '#DC2626', padding: '12px 0' }}>{err}</div>
  }

  if (loaded && kps.length === 0) {
    return (
      <div style={{ fontSize: '12px', color: COLORS.textMuted, padding: '12px', background: COLORS.domainBg, borderRadius: '8px' }}>
        暂无「{subject} · {grade}」的课标知识点数据（目前仅录入了部分学科年级，可不勾选直接创建，AI 将自行规划难度）
      </div>
    )
  }

  // 按领域分组
  const grouped: Record<string, CurriculumKP[]> = {}
  for (const kp of kps) {
    const d = kp.domain || '其他'
    if (!grouped[d]) grouped[d] = []
    grouped[d].push(kp)
  }

  const toggle = (code: string) => {
    if (selectedCodes.includes(code)) {
      onChange(selectedCodes.filter(c => c !== code))
    } else {
      onChange([...selectedCodes, code])
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={{ fontSize: '12px', color: COLORS.textSecondary }}>
          共 {kps.length} 个知识点 · 已选 {selectedCodes.length} 个
        </span>
        {selectedCodes.length > 0 && (
          <button onClick={() => onChange([])} style={{
            background: 'none', border: 'none', fontSize: '12px', color: COLORS.primary, cursor: 'pointer',
          }}>清空</button>
        )}
      </div>

      {Object.keys(grouped).map(domain => (
        <div key={domain} style={{ background: COLORS.domainBg, borderRadius: '10px', padding: '10px 12px' }}>
          <div style={{ fontSize: '12px', fontWeight: 700, color: COLORS.textSecondary, marginBottom: '8px' }}>{domain}</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            {grouped[domain].map(kp => {
              const checked = selectedCodes.includes(kp.kp_code)
              const depth = CW_DEPTH_LEVEL_CONFIG[kp.depth_level] || CW_DEPTH_LEVEL_CONFIG[2]
              return (
                <div key={kp.kp_code} onClick={() => toggle(kp.kp_code)} style={{
                  display: 'flex', alignItems: 'flex-start', gap: '8px', padding: '8px 10px',
                  borderRadius: '8px', cursor: 'pointer',
                  border: `1px solid ${checked ? COLORS.primary : COLORS.border}`,
                  background: checked ? 'rgba(124,58,237,0.05)' : '#fff',
                }}>
                  <input type="checkbox" checked={checked} readOnly style={{ marginTop: '3px', cursor: 'pointer', accentColor: COLORS.primary }} />
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                      <span style={{ fontSize: '14px', fontWeight: 600, color: COLORS.textPrimary }}>{kp.kp_name}</span>
                      <span style={{ padding: '1px 8px', borderRadius: '8px', fontSize: '11px', color: depth.color, background: depth.bg }}>
                        {depth.label}
                      </span>
                    </div>
                    {kp.academic_requirement && (
                      <div style={{ fontSize: '12px', color: COLORS.textMuted, marginTop: '3px', lineHeight: 1.5 }}>
                        {kp.academic_requirement}
                      </div>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      ))}
    </div>
  )
}
