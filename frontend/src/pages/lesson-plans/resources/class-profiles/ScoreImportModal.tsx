import { useState, useCallback, useRef } from 'react'
import type { CSSProperties } from 'react'
import { importClassScores } from '@/api/class-profiles'
import type { ImportScoreRow, ImportScoresResult } from '@/api/class-profiles'

/**
 * ScoreImportModal.tsx — 成绩单导入弹窗（差异化教学·班级学情 批次2b / 2b-2 / 2c）
 *
 * 设计目标（面向不懂技术的老师）：全程只接触 Excel——填考试信息 → 下载模板 → 在 Excel 填 → 上传。
 *
 * 2b-2 关键调整（贴合"一次考试一个批次、对着成绩挨个学生填一行"的真实用法）：
 *   - 考试名称 + 考试日期在弹窗里统一填一次（这一批成绩都归到这次考试），不进 Excel 表。
 *   - Excel 模板四列：学号代号 | 分数 | 薄弱点 | 备注（每行一个学生，逐生不同的内容）。
 *   - 老师对着成绩顺手把这次观察到的薄弱点和备注一起填，免得导完成绩再回平台一个个补。
 *
 * 2c 新增（决策4 第二入口）：导入成功后，结果条里多一个「🧠 顺便让 AI 总结学情」按钮——
 *   刚导完成绩正是数据最新、最想看 AI 解读的时刻。点击关本弹窗并通过 onRequestSummary
 *   通知父页面触发 AI 总结流程（父页面 ClassStudentsPage 负责刷新+弹预览窗）。
 *   onRequestSummary 为可选 prop：父页面不传则不显示该按钮（保持组件独立可复用）。
 *
 * 归并语义（后端）：
 *   - 成绩 → 追加进各学生 scores 数组（看趋势，去重键=考试名+日期）；
 *   - 薄弱点/备注 → 非空则覆盖该生当前值（取最新判断），留空不动；
 *   - 学号不存在则后端自动新建学生（等老师后续手动定 ABC 分层）。
 *
 * ⚠ 合规红线（界面化）：
 *   成绩单（含薄弱点/备注）里请用「学号代号」，不要带学生真实姓名与隐私。
 *   导入只保存学号代号 + 成绩 + 薄弱点/备注，绝不发送给 AI（注入 AI 的只有班级匿名群体结论）。
 *
 * 实现要点：
 *   - 解析/生成 Excel 用 SheetJS(xlsx)，全程【动态 import】不进主包（复用 BatchImportUsersModal 范式）。
 *   - 每行本地预校验（学号非空 + 分数是数字；薄弱点/备注可空），问题行标红；无问题才可提交。
 *   - 提交时校验弹窗里的考试名/日期非空。
 *   - 导入成功后展示后端统计（写入 X 人 Y 条 / 新建 Z 人 / 更新画像 W 人 / 跳过 N 行）。
 *
 * 视觉范式对齐 ClassStudentsPage（同一套 C 配色 / ModalShell 风格）。
 */

const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  border: '#E5E7EB', white: '#FFFFFF', bg: '#F9FAFB',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
  danger: '#EF4444', dangerLight: '#FEF2F2', dangerBorder: '#FECACA',
  success: '#16A34A', successLight: '#F0FDF4',
  // 2c：AI 总结按钮紫色系，与 ClassStudentsPage 一致
  ai: '#7C3AED', aiLight: 'rgba(124,58,237,0.08)',
}

// ==================== 模板列定义（2b-2：四列，考试名/日期不进表）====================
const TEMPLATE_HEADERS = ['学号代号', '分数', '薄弱点', '备注']
const TEMPLATE_SAMPLE = [
  ['01', 85, '函数应用', '需多关注'],
  ['02', 78, '计算粗心', '进步明显'],
  ['03', 92, '', '课堂积极'],
]

const MAX_ROWS = 2000

function todayYMD(): string {
  const d = new Date()
  const m = d.getMonth() + 1
  const day = d.getDate()
  return `${d.getFullYear()}-${m < 10 ? '0' + m : m}-${day < 10 ? '0' + day : day}`
}

function parseScore(raw: unknown): number | null {
  if (raw === null || raw === undefined || raw === '') return null
  const n = typeof raw === 'number' ? raw : Number(String(raw).trim())
  if (!isFinite(n)) return null
  return n
}

interface PreviewRow {
  index: number
  studentCode: string
  score: number | null
  weakTopics: string
  note: string
  localError: string
}

// ---------- 共享样式 ----------
const btnGhost: CSSProperties = { padding: '8px 16px', background: C.white, color: C.textSecondary, border: '1px solid ' + C.border, borderRadius: 8, fontSize: 13, cursor: 'pointer' }
const inputStyle: CSSProperties = { width: '100%', padding: '8px 10px', border: '1px solid ' + C.border, borderRadius: 8, fontSize: 13, color: C.textPrimary, outline: 'none', boxSizing: 'border-box' }
const labelStyle: CSSProperties = { fontSize: 12, fontWeight: 600, color: C.textSecondary, marginBottom: 4, display: 'block' }

export default function ScoreImportModal({
  classProfileId, className, onClose, onImported, onRequestSummary,
}: {
  classProfileId: string
  className: string
  onClose: () => void
  /** 导入成功（有实际写入）后回调，让父页面刷新学生列表 */
  onImported: (result: ImportScoresResult) => void
  /** 2c：可选——导入成功后老师点「顺便让 AI 总结」时回调，父页面负责关窗后触发总结流程。不传则不显示该按钮。 */
  onRequestSummary?: () => void
}) {
  const [examName, setExamName] = useState('')
  const [examDate, setExamDate] = useState(todayYMD())

  const [rows, setRows] = useState<PreviewRow[]>([])
  const [fileName, setFileName] = useState('')
  const [parsing, setParsing] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [downloading, setDownloading] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<ImportScoresResult | null>(null)
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  const buildPreview = useCallback((raw: unknown[][]): PreviewRow[] => {
    return raw.map((cols, i) => {
      const index = i + 1
      const studentCode = String((cols?.[0] ?? '')).trim()
      const score = parseScore(cols?.[1])
      const weakTopics = String((cols?.[2] ?? '')).trim()
      const note = String((cols?.[3] ?? '')).trim()

      let localError = ''
      if (!studentCode) localError = '学号代号为空'
      else if (score === null) localError = '分数为空或不是数字'

      return { index, studentCode, score, weakTopics, note, localError }
    })
  }, [])

  const handleDownloadTemplate = useCallback(async () => {
    try {
      setDownloading(true); setError('')
      const XLSX = await import('xlsx')
      const aoa = [TEMPLATE_HEADERS, ...TEMPLATE_SAMPLE]
      const ws = XLSX.utils.aoa_to_sheet(aoa)
      ws['!cols'] = [{ wch: 12 }, { wch: 8 }, { wch: 18 }, { wch: 18 }]
      const wb = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(wb, ws, '成绩单')
      XLSX.writeFile(wb, `成绩单导入模板.xlsx`)
    } catch (e: any) {
      setError(e?.message ? `生成模板失败：${e.message}` : '生成模板失败')
    } finally {
      setDownloading(false)
    }
  }, [])

  const handleFileSelected = useCallback(async (file: File) => {
    setError(''); setResult(null); setRows([]); setFileName(file.name); setParsing(true)
    try {
      const XLSX = await import('xlsx')
      const buf = await file.arrayBuffer()
      const wb = XLSX.read(buf, { type: 'array' })
      const firstSheet = wb.SheetNames[0]
      if (!firstSheet) { setError('文件中没有任何工作表'); return }
      const ws = wb.Sheets[firstSheet]
      const matrix = XLSX.utils.sheet_to_json<unknown[]>(ws, { header: 1, defval: '', raw: true })
      if (!matrix.length) { setError('表格为空，请先填写后再上传'); return }

      const firstCell = String((matrix[0]?.[0] ?? '')).trim()
      const looksLikeHeader = firstCell === TEMPLATE_HEADERS[0] || firstCell.includes('学号')
      const dataRows = looksLikeHeader ? matrix.slice(1) : matrix

      const nonEmpty = dataRows.filter(cols =>
        String((cols?.[0] ?? '')).trim() ||
        (cols?.[1] !== '' && cols?.[1] !== undefined && cols?.[1] !== null) ||
        String((cols?.[2] ?? '')).trim() ||
        String((cols?.[3] ?? '')).trim()
      )
      if (!nonEmpty.length) {
        setError('未读取到任何数据，请检查是否填在了前四列（学号代号/分数/薄弱点/备注）')
        return
      }
      if (nonEmpty.length > MAX_ROWS) {
        setError(`单次最多导入 ${MAX_ROWS} 行，当前 ${nonEmpty.length} 行，请分批导入`)
        return
      }
      setRows(buildPreview(nonEmpty))
    } catch (e: any) {
      setError(e?.message ? `解析失败：${e.message}` : '解析失败，请确认上传的是 Excel(.xlsx) 文件')
    } finally {
      setParsing(false)
    }
  }, [buildPreview])

  const onInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0]
    if (f) handleFileSelected(f)
    e.target.value = ''
  }

  const localErrorCount = rows.filter(r => r.localError).length
  const examReady = !!examName.trim() && !!examDate.trim()
  const canSubmit = rows.length > 0 && localErrorCount === 0 && examReady && !submitting && !parsing

  const handleSubmit = useCallback(async () => {
    if (!examName.trim()) { setError('请先填写本批「考试名称」'); return }
    if (!examDate.trim()) { setError('请先选择本批「考试日期」'); return }
    if (rows.length === 0) { setError('请先上传并解析成绩单'); return }
    if (localErrorCount > 0) { setError(`有 ${localErrorCount} 行存在问题，请修正 Excel 后重新上传`); return }
    setError(''); setSubmitting(true)
    try {
      const payload: ImportScoreRow[] = rows.map(r => ({
        student_code: r.studentCode,
        score: r.score as number,
        weak_topics: r.weakTopics,
        note: r.note,
      }))
      const res = await importClassScores(classProfileId, {
        exam_name: examName.trim(),
        exam_date: examDate.trim(),
        rows: payload,
      })
      setResult(res)
      if (res.appended_scores > 0 || res.created_students > 0 || res.profile_updated > 0) {
        onImported(res)
      }
    } catch (e: any) {
      setError(e?.message || '导入失败，请稍后重试')
    } finally {
      setSubmitting(false)
    }
  }, [examName, examDate, rows, localErrorCount, classProfileId, onImported])

  // 2c：点「顺便让 AI 总结」——先关本窗，再通知父页面触发总结
  const handleRequestSummary = () => {
    onClose()
    onRequestSummary?.()
  }

  return (
    <div
      onClick={(e) => { if (e.target === e.currentTarget && !submitting) onClose() }}
      style={{ position: 'fixed', inset: 0, zIndex: 9000, background: 'rgba(17,24,39,0.45)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 20 }}>
      <div style={{ background: C.white, borderRadius: 16, width: 780, maxWidth: '95vw', maxHeight: '90vh', overflow: 'hidden', display: 'flex', flexDirection: 'column', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>

        {/* 头部 */}
        <div style={{ padding: '18px 22px', borderBottom: '1px solid ' + C.border, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ fontSize: 16, fontWeight: 700, color: C.textPrimary }}>📥 导入成绩单 · {className}</div>
          <button onClick={() => { if (!submitting) onClose() }} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: 20, color: C.textMuted }}>×</button>
        </div>

        {/* 内容（可滚动） */}
        <div style={{ padding: 22, overflowY: 'auto', flex: 1 }}>

          {/* 合规红线提示带 */}
          <div style={{ display: 'flex', gap: 10, alignItems: 'flex-start', background: C.dangerLight, border: '1px solid ' + C.dangerBorder, borderRadius: 10, padding: '12px 14px', marginBottom: 16 }}>
            <span style={{ fontSize: 15 }}>🔒</span>
            <div style={{ fontSize: 12.5, color: '#991B1B', lineHeight: 1.7 }}>
              <strong>隐私红线：</strong>表格里（含薄弱点、备注）请用<strong>学号代号</strong>（如 01 / S001），
              <strong>不要带学生真实姓名或隐私信息</strong>。导入只保存学号代号与学情判断，绝不发送给 AI。
            </div>
          </div>

          {/* 导入成功结果（出结果后只显示这块）*/}
          {result ? (
            <div style={{ padding: '14px 16px', borderRadius: 10, marginBottom: 4, background: C.successLight, border: '1px solid #BBF7D0' }}>
              <div style={{ fontSize: 14, fontWeight: 700, color: C.success, marginBottom: 6 }}>✓ 导入完成</div>
              <div style={{ fontSize: 13, color: C.textPrimary, lineHeight: 1.8 }}>
                本批考试「<b>{examName}</b> · {examDate}」共处理 <b>{result.total_rows}</b> 行：
                为 <b>{result.affected_students}</b> 名学生写入/更新了 <b>{result.appended_scores}</b> 条成绩
                {result.created_students > 0 && <>，新建 <b>{result.created_students}</b> 名学生档案</>}
                {result.profile_updated > 0 && <>，更新 <b>{result.profile_updated}</b> 名学生的薄弱点/备注</>}
                {result.skipped_rows > 0 && <>，<span style={{ color: C.danger }}>跳过 {result.skipped_rows} 行</span></>}。
              </div>
              {result.errors && result.errors.length > 0 && (
                <div style={{ marginTop: 8, fontSize: 12, color: C.danger, lineHeight: 1.7 }}>
                  {result.errors.map((er, i) => <div key={i}>· {er}</div>)}
                </div>
              )}
              {/* 2c：导入后 AI 总结快捷入口（onRequestSummary 存在才显示）+ 完成按钮 */}
              <div style={{ marginTop: 12, display: 'flex', gap: 10, flexWrap: 'wrap' }}>
                {onRequestSummary && (
                  <button onClick={handleRequestSummary}
                    style={{ padding: '8px 18px', background: C.aiLight, color: C.ai, border: '1px solid ' + C.ai, borderRadius: 8, fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>
                    🧠 顺便让 AI 总结学情
                  </button>
                )}
                <button onClick={onClose} style={{ padding: '8px 18px', background: C.success, color: '#fff', border: 'none', borderRadius: 8, fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>完成，返回查看成绩</button>
              </div>
              {onRequestSummary && (
                <div style={{ marginTop: 8, fontSize: 11.5, color: C.textMuted, lineHeight: 1.6 }}>
                  💡 刚导入的成绩已计入。点「让 AI 总结学情」会基于全班最新的匿名统计，生成可写回班级卡的学情画像。
                </div>
              )}
            </div>
          ) : (
            <>
              {/* 本批考试信息（整批统一，2b-2）*/}
              <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
                <div style={{ flex: 2 }}>
                  <label style={labelStyle}>本批考试名称 <span style={{ color: C.danger }}>*</span></label>
                  <input value={examName} onChange={(e) => setExamName(e.target.value)} style={inputStyle}
                    placeholder="如：3月月考 / 第2次周考 / 期中考试" />
                </div>
                <div style={{ flex: 1 }}>
                  <label style={labelStyle}>本批考试日期 <span style={{ color: C.danger }}>*</span></label>
                  <input type="date" value={examDate} onChange={(e) => setExamDate(e.target.value)} style={inputStyle} />
                </div>
              </div>

              {/* 三步引导 */}
              <div style={{ background: C.primaryLight, borderRadius: 10, padding: '12px 14px', marginBottom: 16, fontSize: 12.5, color: C.primary, lineHeight: 1.8 }}>
                <div style={{ fontWeight: 700, marginBottom: 2 }}>操作三步：</div>
                ① 上方填好「这次是哪场考试、哪天考的」（整批统一，表格里不用再填）<br />
                ② 点「下载 Excel 模板」，照示例填：每行一个学生，四列是 <b>学号代号 / 分数 / 薄弱点 / 备注</b>（薄弱点、备注没有可留空）<br />
                ③ 上传填好的 Excel。<b>学号不存在会自动建档</b>；薄弱点/备注填了就<b>覆盖更新</b>该生当前记录，留空则保留原有。
              </div>

              {/* 全局错误 */}
              {error && (
                <div style={{ padding: '10px 14px', borderRadius: 8, marginBottom: 14, background: C.dangerLight, color: C.danger, fontSize: 13, lineHeight: 1.6 }}>{error}</div>
              )}

              {/* 下载模板 + 上传 */}
              <div style={{ display: 'flex', gap: 10, marginBottom: 16, flexWrap: 'wrap' }}>
                <button onClick={handleDownloadTemplate} disabled={downloading}
                  style={{ padding: '9px 16px', borderRadius: 9, border: '1px solid ' + C.primary, background: C.primaryLight, color: C.primary, fontSize: 13, fontWeight: 600, cursor: downloading ? 'wait' : 'pointer' }}>
                  {downloading ? '生成中…' : '⬇️ 下载 Excel 模板'}
                </button>
                <button onClick={() => fileInputRef.current?.click()} disabled={parsing || submitting}
                  style={{ padding: '9px 16px', borderRadius: 9, border: 'none', background: C.primary, color: '#fff', fontSize: 13, fontWeight: 600, cursor: (parsing || submitting) ? 'wait' : 'pointer' }}>
                  {parsing ? '解析中…' : '📂 上传填好的 Excel'}
                </button>
                <input ref={fileInputRef} type="file"
                  accept=".xlsx,.xls,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.ms-excel"
                  style={{ display: 'none' }} onChange={onInputChange} />
                {fileName && <div style={{ display: 'flex', alignItems: 'center', fontSize: 12, color: C.textSecondary }}>已选：{fileName}</div>}
              </div>

              {/* 预览表（四列）*/}
              {rows.length > 0 && (
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8, flexWrap: 'wrap' }}>
                    <span style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>预览（共 {rows.length} 行）</span>
                    {examName.trim() && (
                      <span style={{ fontSize: 12, color: C.primary, background: C.primaryLight, padding: '2px 10px', borderRadius: 999 }}>
                        本批：{examName.trim()} · {examDate}
                      </span>
                    )}
                    {localErrorCount > 0
                      ? <span style={{ fontSize: 12, color: C.danger }}>⚠️ {localErrorCount} 行有问题，需修正</span>
                      : <span style={{ fontSize: 12, color: C.success }}>✓ 校验通过，可导入</span>}
                  </div>
                  <div style={{ border: '1px solid ' + C.border, borderRadius: 10, overflow: 'hidden' }}>
                    <div style={{ display: 'grid', gridTemplateColumns: '40px 1fr 0.7fr 1.3fr 1.3fr 1.3fr', padding: '9px 12px', background: C.bg, borderBottom: '1px solid ' + C.border, fontSize: 12, fontWeight: 600, color: C.textSecondary }}>
                      <span>#</span><span>学号代号</span><span>分数</span><span>薄弱点</span><span>备注</span><span>校验</span>
                    </div>
                    <div style={{ maxHeight: 280, overflowY: 'auto' }}>
                      {rows.map((r, idx) => {
                        const hasErr = !!r.localError
                        return (
                          <div key={idx} style={{ display: 'grid', gridTemplateColumns: '40px 1fr 0.7fr 1.3fr 1.3fr 1.3fr', padding: '8px 12px', fontSize: 12, alignItems: 'center', borderBottom: idx < rows.length - 1 ? '1px solid ' + C.border : 'none', background: hasErr ? C.dangerLight : 'transparent' }}>
                            <span style={{ color: C.textMuted }}>{r.index}</span>
                            <span style={{ color: C.textPrimary, fontWeight: 600 }}>{r.studentCode || <em style={{ color: C.textMuted }}>（空）</em>}</span>
                            <span style={{ color: C.textSecondary }}>{r.score === null ? <em style={{ color: C.textMuted }}>（空）</em> : r.score}</span>
                            <span style={{ color: C.textSecondary }}>{r.weakTopics || <span style={{ color: C.textMuted }}>—</span>}</span>
                            <span style={{ color: C.textSecondary }}>{r.note || <span style={{ color: C.textMuted }}>—</span>}</span>
                            <span style={{ color: hasErr ? C.danger : C.success, fontWeight: hasErr ? 600 : 400 }}>{hasErr ? r.localError : '✓'}</span>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        {/* 底部操作栏（仅未出结果时显示） */}
        {!result && (
          <div style={{ padding: '14px 22px', borderTop: '1px solid ' + C.border, display: 'flex', gap: 10 }}>
            <button onClick={() => { if (!submitting) onClose() }} style={{ ...btnGhost, flex: 1, textAlign: 'center' }}>取消</button>
            <button onClick={handleSubmit} disabled={!canSubmit}
              style={{ flex: 2, padding: '10px', borderRadius: 9, border: 'none', background: canSubmit ? C.primary : C.textMuted, color: '#fff', fontSize: 14, fontWeight: 600, cursor: canSubmit ? 'pointer' : 'not-allowed' }}>
              {submitting ? '导入中…'
                : !examName.trim() ? '请先填考试名称'
                : rows.length > 0 ? `确认导入 ${rows.length} 条成绩` : '请先上传成绩单'}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
