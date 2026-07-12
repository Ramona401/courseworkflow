/**
 * BatchImportUsersModal.tsx — 批量导入教师弹窗（Phase 6.4 / 合并重构 / 角色批次入口改）
 *
 * 设计目标（面向不懂技术的老师）：
 *   全程只接触 Excel，三步走——下载 Excel 模板 → 在 Excel 里填 → 上传 Excel。
 *
 * 【本次改动：角色批次入口】（与系统操作员确认）
 *   背景：同事反馈「Excel 里没有角色，担心搞错」。讨论后确定的最优解不是把角色塞进
 *   Excel 让老师逐行填（易错、需数据验证、移动端兼容差），而是——
 *     · Excel 永远保持干净三列（用户名 / 姓名 / 密码），角色不进表；
 *     · 角色由「这一批是哪类账号」在上传前明确选定，整批统一。
 *   因此把原来朴素的「统一角色下拉」升级为醒目的【角色批次卡片】：
 *     [ 骨干教师批次(operator) ]　[ 普通教师批次(viewer) ]
 *   选哪张卡，这一整批就全部建为该角色，文案明确强调「本批将全部建为 XX」。
 *   这等价于「每个角色一张表分开传」，但无需维护多份模板文件，体验更省事更不易错。
 *
 *   ⚠️ 角色范围：批量导入后端白名单只允许 operator(骨干教师) / viewer(普通教师) 两类，
 *   刻意不含 senior_operator / region_admin / admin 等带权限角色——管理员账号应由
 *   系统管理员在「新建用户」入口逐个单独创建，不走批量（安全设计，后端二次强制）。
 *
 * 合并重构改动（此前）：
 *   - 批量接口统一走 admin 的 batchCreateAdminUsers（/api/v1/admin/users/batch）。
 *   - props：
 *       * mode='admin'：系统管理员视角——弹窗内顶部显示「目标学校」下拉（必选），
 *         选项来自 schools；提交时随请求体带 school_id。未选学校则拦截提示。
 *       * mode='self' ：学校管理员(senior)视角——不显示学校下拉，
 *         不带 school_id，后端 resolveSchoolScope 强制本校。
 *       * schools：admin 模式下的学校下拉数据源（{id,name}[]），由 AdminPage 传入。
 *
 * 实现要点：
 *   - 解析/生成 Excel 用 SheetJS(xlsx)，全部走【动态 import】，不进主包。
 *   - 解析后逐行本地预校验（空字段/密码<6位/批内重复），问题行标红。
 *   - 提交 batchCreateAdminUsers（整批回滚 + 行号失败明细）：
 *       success=true  → 成功提示后回调刷新+关闭；
 *       success=false → 把后端 failures 按行号合并回预览表逐行红字，改完可重传。
 *   - 角色为批次级统一角色（operator/viewer），由角色批次卡片选定。
 *
 * Excel 模板列：登录用户名 | 教师姓名 | 初始密码（按列位置映射，首行表头自动跳过）。
 */
import { useState, useCallback, useRef } from 'react'
import { C } from './adminConstants'

/** 学校下拉项（admin 模式用） */
interface SchoolOption {
  id: string
  name: string
}

interface BatchImportUsersModalProps {
  onClose: () => void
  /** 成功导入后的回调（刷新列表用），参数为成功创建的人数 */
  onImported: (createdCount: number) => void
  /**
   * 视角模式（合并重构新增）：
   *   - 'admin'：系统管理员，弹窗内选目标学校，请求体带 school_id；
   *   - 'self' ：学校管理员(senior)，不选学校，后端强制本校。
   */
  mode: 'admin' | 'self'
  /** admin 模式下的学校下拉数据源（self 模式忽略） */
  schools?: SchoolOption[]
}

/** 预览行：解析自 Excel 的一行 + 本地/后端校验状态 */
interface PreviewRow {
  index: number            // 行号（1-based，对齐老师在 Excel 看到的行，已扣除表头）
  username: string
  displayName: string
  password: string
  localError: string       // 本地预校验错误，空串=本地通过
  serverError: string      // 后端返回的该行失败原因，空串=无
}

// ==================== Excel 模板列定义 ====================
// 模板保持干净三列，角色不进 Excel（由角色批次卡片在上传前选定）
const TEMPLATE_HEADERS = ['登录用户名', '教师姓名', '初始密码']
const TEMPLATE_SAMPLE = [
  ['teacher01', '张老师', '123456'],
  ['teacher02', '李老师', '123456'],
]

// ==================== 角色批次定义 ====================
// 与后端批量白名单一致：仅 operator / viewer 两类。
// 管理员等带权限角色不走批量（应在「新建用户」逐个单独建）。
type BatchRoleValue = 'operator' | 'viewer'
interface RoleBatchDef {
  value: BatchRoleValue
  title: string            // 卡片主标题
  subtitle: string         // 卡片副标题（角色英文名，给懂的人对照）
  desc: string             // 一句话说明这类账号是什么
  emoji: string
}
const ROLE_BATCHES: RoleBatchDef[] = [
  {
    value: 'operator',
    title: '骨干教师',
    subtitle: 'operator',
    desc: '可备课、可参与教研组管理等',
    emoji: '⭐',
  },
  {
    value: 'viewer',
    title: '普通教师',
    subtitle: 'viewer',
    desc: '可备课、查看共享内容',
    emoji: '👤',
  },
]

export function BatchImportUsersModal({ onClose, onImported, mode, schools = [] }: BatchImportUsersModalProps) {
  // 角色批次：默认普通教师（最常见、最小权限，更安全的默认）
  const [role, setRole] = useState<BatchRoleValue>('viewer')
  // admin 模式：弹窗内选中的目标学校 id（self 模式不使用）
  const [selectedSchoolId, setSelectedSchoolId] = useState('')
  const [rows, setRows] = useState<PreviewRow[]>([])
  const [fileName, setFileName] = useState('')
  const [parsing, setParsing] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [downloading, setDownloading] = useState(false)
  const [error, setError] = useState('')
  const [doneMsg, setDoneMsg] = useState('')
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  // 当前选中角色批次的展示信息（用于各处文案）
  const currentRole = ROLE_BATCHES.find(r => r.value === role) ?? ROLE_BATCHES[1]

  // ==================== 本地预校验 ====================
  const validateRows = useCallback((raw: { username: string; displayName: string; password: string }[]): PreviewRow[] => {
    const seen = new Map<string, number>()
    return raw.map((r, i) => {
      const index = i + 1
      const username = r.username.trim()
      const displayName = r.displayName.trim()
      const password = r.password
      let localError = ''
      if (!username) {
        localError = '用户名不能为空'
      } else if (!displayName) {
        localError = '教师姓名不能为空'
      } else if (!password || password.length < 6) {
        localError = '初始密码至少6位'
      } else if (seen.has(username)) {
        localError = `与第${seen.get(username)}行用户名重复`
      }
      if (username && !seen.has(username)) seen.set(username, index)
      return { index, username, displayName, password, localError, serverError: '' }
    })
  }, [])

  // ==================== 下载 Excel 模板 ====================
  const handleDownloadTemplate = useCallback(async () => {
    try {
      setDownloading(true)
      setError('')
      const XLSX = await import('xlsx')
      const aoa = [TEMPLATE_HEADERS, ...TEMPLATE_SAMPLE]
      const ws = XLSX.utils.aoa_to_sheet(aoa)
      ws['!cols'] = [{ wch: 18 }, { wch: 14 }, { wch: 14 }]
      const wb = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(wb, ws, '教师名单')
      XLSX.writeFile(wb, '教师批量导入模板.xlsx')
    } catch (e: unknown) {
      setError(e instanceof Error ? `生成模板失败：${e.message}` : '生成模板失败')
    } finally {
      setDownloading(false)
    }
  }, [])

  // ==================== 解析上传的 Excel ====================
  const handleFileSelected = useCallback(async (file: File) => {
    setError('')
    setDoneMsg('')
    setRows([])
    setFileName(file.name)
    setParsing(true)
    try {
      const XLSX = await import('xlsx')
      const buf = await file.arrayBuffer()
      const wb = XLSX.read(buf, { type: 'array' })
      const firstSheet = wb.SheetNames[0]
      if (!firstSheet) {
        setError('文件中没有任何工作表')
        return
      }
      const ws = wb.Sheets[firstSheet]
      const matrix = XLSX.utils.sheet_to_json<unknown[]>(ws, { header: 1, defval: '' })
      if (!matrix.length) {
        setError('表格为空，请先填写后再上传')
        return
      }
      const firstCell = String((matrix[0]?.[0] ?? '')).trim()
      const looksLikeHeader =
        firstCell === TEMPLATE_HEADERS[0] || firstCell.includes('用户名') || firstCell.includes('账号')
      const dataRows = looksLikeHeader ? matrix.slice(1) : matrix
      const raw = dataRows
        .map(cols => ({
          username: String((cols?.[0] ?? '')).trim(),
          displayName: String((cols?.[1] ?? '')).trim(),
          password: String((cols?.[2] ?? '')).trim(),
        }))
        .filter(r => r.username || r.displayName || r.password)
      if (!raw.length) {
        setError('未读取到任何教师数据，请检查是否填在了前三列（用户名/姓名/密码）')
        return
      }
      if (raw.length > 200) {
        setError(`单次最多导入 200 人，当前 ${raw.length} 人，请分批导入`)
        return
      }
      setRows(validateRows(raw))
    } catch (e: unknown) {
      setError(e instanceof Error ? `解析失败：${e.message}` : '解析失败，请确认上传的是 Excel(.xlsx) 文件')
    } finally {
      setParsing(false)
    }
  }, [validateRows])

  const onInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0]
    if (f) handleFileSelected(f)
    e.target.value = ''
  }

  // ==================== 提交批量创建 ====================
  const localErrorCount = rows.filter(r => r.localError).length
  // admin 模式必须先选学校；self 模式无此要求
  const needSchool = mode === 'admin'
  const schoolReady = !needSchool || !!selectedSchoolId
  const canSubmit = rows.length > 0 && localErrorCount === 0 && schoolReady && !submitting && !parsing

  const handleSubmit = useCallback(async () => {
    if (rows.length === 0) { setError('请先上传并解析教师名单'); return }
    if (localErrorCount > 0) { setError(`有 ${localErrorCount} 行存在问题，请修正 Excel 后重新上传`); return }
    if (needSchool && !selectedSchoolId) { setError('请先在上方选择目标学校'); return }
    setError('')
    setDoneMsg('')
    setSubmitting(true)
    try {
      // 合并重构：统一走 /admin/users/batch
      // admin 模式带 school_id（弹窗内所选）；self 模式不带，后端强制本校
      // role 取自上方选定的角色批次，整批统一
      const { batchCreateAdminUsers } = await import('@/api/admin')
      const result = await batchCreateAdminUsers({
        role,
        ...(needSchool ? { school_id: selectedSchoolId } : {}),
        users: rows.map(r => ({
          username: r.username,
          display_name: r.displayName,
          password: r.password,
        })),
      })
      if (result.success) {
        setDoneMsg(`✓ 成功导入 ${result.created_count} 位${currentRole.title}`)
        setTimeout(() => { onImported(result.created_count); onClose() }, 900)
      } else {
        const failMap = new Map<number, string>()
        for (const f of result.failures || []) failMap.set(f.index, f.reason)
        setRows(prev => prev.map(r => ({ ...r, serverError: failMap.get(r.index) || '' })))
        setError(`导入未成功：有 ${result.failures?.length || 0} 行存在问题（整批未创建任何用户），请按下方红字修正 Excel 后重新上传`)
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '批量导入失败，请稍后重试')
    } finally {
      setSubmitting(false)
    }
  }, [rows, role, currentRole.title, localErrorCount, needSchool, selectedSchoolId, onImported, onClose])

  // ==================== 渲染 ====================
  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 10000,
        background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}
      onClick={e => { if (e.target === e.currentTarget && !submitting) onClose() }}>
      <div style={{ background: C.white, borderRadius: '20px', width: '720px', maxWidth: '94vw', maxHeight: '90vh', overflow: 'hidden', boxShadow: '0 20px 60px rgba(0,0,0,0.2)', display: 'flex', flexDirection: 'column' }}>

        {/* 头部 */}
        <div style={{ padding: '20px 24px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0 }}>
          <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>📥 批量导入教师</div>
          <button onClick={() => { if (!submitting) onClose() }} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>×</button>
        </div>

        {/* 内容（可滚动） */}
        <div style={{ padding: '24px', overflowY: 'auto', flex: 1 }}>

          {/* 三步引导 */}
          <div style={{ background: C.primaryLight, borderRadius: '10px', padding: '14px 16px', marginBottom: '18px', fontSize: '13px', color: C.primary, lineHeight: 1.8 }}>
            <div style={{ fontWeight: 700, marginBottom: '4px' }}>操作三步：</div>
            ① 点下方「下载 Excel 模板」，用 Excel/WPS 打开<br />
            ② 照着示例填写：每行一位教师，三列分别是 <b>登录用户名 / 教师姓名 / 初始密码</b><br />
            ③ 选好「这一批是哪类账号」，再把填好的 Excel 上传回来即可
          </div>

          {/* 全局错误 / 成功提示 */}
          {error && (
            <div style={{ padding: '10px 14px', borderRadius: '8px', marginBottom: '16px', background: C.dangerLight, color: C.danger, fontSize: '13px', lineHeight: 1.6 }}>
              {error}
            </div>
          )}
          {doneMsg && (
            <div style={{ padding: '10px 14px', borderRadius: '8px', marginBottom: '16px', background: C.successLight, color: C.success, fontSize: '13px', fontWeight: 600 }}>
              {doneMsg}
            </div>
          )}

          {/* admin 模式：目标学校选择（必选） */}
          {mode === 'admin' && (
            <div style={{ marginBottom: '16px' }}>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                目标学校 <span style={{ color: C.danger }}>*</span>
              </label>
              <select
                value={selectedSchoolId}
                onChange={e => setSelectedSchoolId(e.target.value)}
                disabled={submitting}
                style={{ width: '100%', padding: '10px 14px', borderRadius: '10px', border: `1px solid ${selectedSchoolId ? C.border : C.danger}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: C.white }}>
                <option value="">请选择要导入到的学校...</option>
                {schools.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
              </select>
              <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '6px' }}>
                💡 本批教师将全部加入所选学校。
              </div>
            </div>
          )}

          {/* ========== 角色批次选择（卡片式，替代原朴素下拉） ========== */}
          <div style={{ marginBottom: '18px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
              这一批是哪类账号？<span style={{ color: C.danger }}>*</span>
            </label>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
              {ROLE_BATCHES.map(rb => {
                const active = role === rb.value
                return (
                  <button
                    key={rb.value}
                    type="button"
                    onClick={() => { if (!submitting) setRole(rb.value) }}
                    disabled={submitting}
                    style={{
                      textAlign: 'left',
                      padding: '14px 16px',
                      borderRadius: '12px',
                      border: active ? `2px solid ${C.primary}` : `1px solid ${C.border}`,
                      background: active ? C.primaryLight : C.white,
                      cursor: submitting ? 'not-allowed' : 'pointer',
                      transition: 'all 0.15s',
                      boxShadow: active ? `0 2px 8px ${C.primaryLight}` : 'none',
                    }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
                      <span style={{ fontSize: '18px' }}>{rb.emoji}</span>
                      <span style={{ fontSize: '14px', fontWeight: 700, color: active ? C.primary : C.text }}>
                        {rb.title}
                      </span>
                      <span style={{ fontSize: '11px', color: C.textMuted, fontFamily: 'monospace' }}>
                        {rb.subtitle}
                      </span>
                      {active && <span style={{ marginLeft: 'auto', fontSize: '13px', color: C.primary, fontWeight: 700 }}>✓</span>}
                    </div>
                    <div style={{ fontSize: '11px', color: C.textSec, lineHeight: 1.5 }}>
                      {rb.desc}
                    </div>
                  </button>
                )
              })}
            </div>
            <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '8px', lineHeight: 1.6 }}>
              💡 本批 <b style={{ color: C.primary }}>全部</b> 建为「{currentRole.title}」。若要导入另一类账号，请填一张新表分批导入。<br />
              管理员等带权限的账号请在「新建用户」里单独创建，不通过批量导入。
            </div>
          </div>

          {/* 下载模板 + 上传文件 两个按钮 */}
          <div style={{ display: 'flex', gap: '10px', marginBottom: '18px', flexWrap: 'wrap' }}>
            <button
              onClick={handleDownloadTemplate}
              disabled={downloading}
              style={{ padding: '10px 18px', borderRadius: '10px', border: `1px solid ${C.primary}`, background: C.primaryLight, color: C.primary, fontSize: '13px', fontWeight: 600, cursor: downloading ? 'wait' : 'pointer' }}>
              {downloading ? '生成中...' : '⬇️ 下载 Excel 模板'}
            </button>
            <button
              onClick={() => fileInputRef.current?.click()}
              disabled={parsing || submitting}
              style={{ padding: '10px 18px', borderRadius: '10px', border: 'none', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: (parsing || submitting) ? 'wait' : 'pointer' }}>
              {parsing ? '解析中...' : '📂 上传填好的 Excel'}
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept=".xlsx,.xls,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.ms-excel"
              style={{ display: 'none' }}
              onChange={onInputChange}
            />
            {fileName && (
              <div style={{ display: 'flex', alignItems: 'center', fontSize: '12px', color: C.textSec }}>
                已选文件：{fileName}
              </div>
            )}
          </div>

          {/* 预览表 */}
          {rows.length > 0 && (
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '8px', flexWrap: 'wrap' }}>
                <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>预览（共 {rows.length} 行）</span>
                {/* 角色批次提示标签，让操作员在提交前再次确认这一批的角色 */}
                <span style={{ fontSize: '12px', fontWeight: 600, color: C.primary, background: C.primaryLight, padding: '2px 10px', borderRadius: '999px' }}>
                  {currentRole.emoji} 本批角色：{currentRole.title}
                </span>
                {localErrorCount > 0
                  ? <span style={{ fontSize: '12px', color: C.danger }}>⚠️ {localErrorCount} 行有问题，需修正</span>
                  : <span style={{ fontSize: '12px', color: C.success }}>✓ 校验通过，可提交</span>}
              </div>
              <div style={{ border: `1px solid ${C.border}`, borderRadius: '10px', overflow: 'hidden' }}>
                <div style={{ display: 'grid', gridTemplateColumns: '48px 1.4fr 1.2fr 1fr 1.6fr', padding: '10px 14px', background: C.bg, borderBottom: `1px solid ${C.border}`, fontSize: '12px', fontWeight: 600, color: C.textSec }}>
                  <span>#</span><span>登录用户名</span><span>教师姓名</span><span>初始密码</span><span>校验</span>
                </div>
                <div style={{ maxHeight: '260px', overflowY: 'auto' }}>
                  {rows.map((r, idx) => {
                    const err = r.serverError || r.localError
                    const hasErr = !!err
                    return (
                      <div key={r.index}
                        style={{ display: 'grid', gridTemplateColumns: '48px 1.4fr 1.2fr 1fr 1.6fr', padding: '9px 14px', fontSize: '12px', alignItems: 'center', borderBottom: idx < rows.length - 1 ? `1px solid ${C.border}` : 'none', background: hasErr ? C.dangerLight : 'transparent' }}>
                        <span style={{ color: C.textMuted }}>{r.index}</span>
                        <span style={{ color: C.text, fontWeight: 500 }}>{r.username || <em style={{ color: C.textMuted }}>（空）</em>}</span>
                        <span style={{ color: C.text }}>{r.displayName || <em style={{ color: C.textMuted }}>（空）</em>}</span>
                        <span style={{ color: C.textSec, fontFamily: 'monospace' }}>{r.password ? '•'.repeat(Math.min(r.password.length, 8)) : <em style={{ color: C.textMuted }}>（空）</em>}</span>
                        <span style={{ color: hasErr ? C.danger : C.success, fontWeight: hasErr ? 600 : 400 }}>
                          {hasErr ? err : '✓'}
                        </span>
                      </div>
                    )
                  })}
                </div>
              </div>
            </div>
          )}
        </div>

        {/* 底部操作栏 */}
        <div style={{ padding: '16px 24px', borderTop: `1px solid ${C.border}`, display: 'flex', gap: '10px', flexShrink: 0 }}>
          <button onClick={() => { if (!submitting) onClose() }}
            style={{ flex: 1, padding: '11px', borderRadius: '10px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '14px', color: C.textSec, cursor: submitting ? 'not-allowed' : 'pointer' }}>
            取消
          </button>
          <button onClick={handleSubmit} disabled={!canSubmit}
            style={{
              flex: 2, padding: '11px', borderRadius: '10px', border: 'none',
              background: canSubmit ? `linear-gradient(135deg,${C.primary},#7C3AED)` : C.textMuted,
              color: '#fff', fontSize: '14px', fontWeight: 600,
              cursor: canSubmit ? 'pointer' : 'not-allowed',
            }}>
            {submitting ? '导入中...' : needSchool && !selectedSchoolId ? '请先选择目标学校' : rows.length > 0 ? `确认导入 ${rows.length} 位${currentRole.title}` : '请先上传名单'}
          </button>
        </div>
      </div>
    </div>
  )
}
