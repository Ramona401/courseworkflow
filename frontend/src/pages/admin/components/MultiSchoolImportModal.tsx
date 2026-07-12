/**
 * MultiSchoolImportModal.tsx — 跨区域多校批量导入教师弹窗（跨校批量·新增）
 *
 * admin 把【一个区域下多所学校】的老师，汇总成一张 Excel 一次性导入。
 * 与单校弹窗(BatchImportUsersModal)并存、互不影响：
 *   - 单校：一次一所学校、整批回滚；
 *   - 跨校(本组件)：一次一个区域多所学校、逐行成败、重名自动改名。
 * 仅 admin 可用（后端 adminOnly + handler 内双保险）。
 *
 * 流程：
 *   生成模板：选区域 → 勾选学校(可全选) → 选角色批次 → 下载 Excel
 *     · Sheet1「教师名单」4列：登录用户名|教师姓名|初始密码|所属学校(尽力加下拉)
 *     · Sheet2「学校清单（请勿修改）」：学校名|学校ID(照抄+上传反查兜底)
 *   上传名单：上传 → 解析4列 → "学校名→ID"反查(内存映射优先,Sheet2兜底) → 预览标红 → 提交
 *     结果：改名清单(一键复制通知) + 失败清单
 *
 * 下拉双保险：SheetJS 写下拉在部分版本/WPS 可能失效；失效时老师照 Sheet2 手填学校名，
 *   靠"学校名→ID"映射反查照样可用；名字对不上则该行标红提示。功能不押注下拉一定生效。
 */
import { useState, useCallback, useRef } from 'react'
import { C } from './adminConstants'
import {
  getAdminOrgs,
  getRegionSchools,
  batchCreateMultiSchoolUsers,
  type RegionSchoolItem,
  type MultiSchoolBatchResult,
} from '@/api/admin'

interface MultiSchoolImportModalProps {
  onClose: () => void
  onImported: (createdCount: number) => void
}

interface RegionOption {
  id: string
  name: string
}

interface PreviewRow {
  index: number
  username: string
  displayName: string
  password: string
  schoolName: string
  schoolId: string
  localError: string
}

const TEMPLATE_HEADERS = ['登录用户名', '教师姓名', '初始密码', '所属学校']
const SCHOOL_SHEET_HEADERS = ['学校名（填表时照此填写）', '学校ID（系统用，请勿修改）']
const MAX_ROWS = 2000

type BatchRoleValue = 'operator' | 'viewer'
interface RoleBatchDef {
  value: BatchRoleValue
  title: string
  subtitle: string
  desc: string
  emoji: string
}
const ROLE_BATCHES: RoleBatchDef[] = [
  { value: 'operator', title: '骨干教师', subtitle: 'operator', desc: '可备课、可参与教研组管理等', emoji: '⭐' },
  { value: 'viewer', title: '普通教师', subtitle: 'viewer', desc: '可备课、查看共享内容', emoji: '👤' },
]

export function MultiSchoolImportModal({ onClose, onImported }: MultiSchoolImportModalProps) {
  const [regions, setRegions] = useState<RegionOption[]>([])
  const [regionsLoaded, setRegionsLoaded] = useState(false)
  const [selectedRegionId, setSelectedRegionId] = useState('')
  const [schools, setSchools] = useState<RegionSchoolItem[]>([])
  const [loadingSchools, setLoadingSchools] = useState(false)
  const [checkedSchoolIds, setCheckedSchoolIds] = useState<Set<string>>(new Set())

  const [role, setRole] = useState<BatchRoleValue>('viewer')

  const nameToIdMapRef = useRef<Map<string, string>>(new Map())

  const [rows, setRows] = useState<PreviewRow[]>([])
  const [fileName, setFileName] = useState('')
  const [parsing, setParsing] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [downloading, setDownloading] = useState(false)
  const [error, setError] = useState('')
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  const [result, setResult] = useState<MultiSchoolBatchResult | null>(null)
  const [copyMsg, setCopyMsg] = useState('')

  const currentRole = ROLE_BATCHES.find(r => r.value === role) ?? ROLE_BATCHES[1]

  const loadRegions = useCallback(async () => {
    if (regionsLoaded) return
    try {
      const orgs = await getAdminOrgs({ type: 'region' })
      setRegions(orgs.map(o => ({ id: o.id, name: o.name })))
      setRegionsLoaded(true)
    } catch {
      setError('加载区域列表失败，请关闭重试')
    }
  }, [regionsLoaded])

  const initedRef = useRef(false)
  if (!initedRef.current) {
    initedRef.current = true
    void loadRegions()
  }

  const handleRegionChange = useCallback(async (regionId: string) => {
    setSelectedRegionId(regionId)
    setSchools([])
    setCheckedSchoolIds(new Set())
    setError('')
    if (!regionId) return
    try {
      setLoadingSchools(true)
      const list = await getRegionSchools(regionId)
      setSchools(list)
      if (list.length === 0) {
        setError('该区域下暂无学校，请先在组织架构里添加学校')
      }
    } catch {
      setError('加载该区域学校失败，请重试')
    } finally {
      setLoadingSchools(false)
    }
  }, [])

  const toggleSchool = useCallback((id: string) => {
    setCheckedSchoolIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  const allChecked = schools.length > 0 && checkedSchoolIds.size === schools.length
  const toggleSelectAll = useCallback(() => {
    setCheckedSchoolIds(prev => {
      if (prev.size === schools.length) return new Set()
      return new Set(schools.map(s => s.id))
    })
  }, [schools])

  const checkedSchools = schools.filter(s => checkedSchoolIds.has(s.id))

  const handleDownloadTemplate = useCallback(async () => {
    if (checkedSchools.length === 0) {
      setError('请先选择至少一所学校，再下载模板')
      return
    }
    try {
      setDownloading(true)
      setError('')
      const XLSX = await import('xlsx')

      const map = new Map<string, string>()
      for (const s of checkedSchools) map.set(s.name.trim(), s.id)
      nameToIdMapRef.current = map

      const sampleSchool = checkedSchools[0]?.name ?? ''
      const sheet1Data = [
        TEMPLATE_HEADERS,
        ['teacher01', '张老师', '123456', sampleSchool],
        ['teacher02', '李老师', '123456', sampleSchool],
      ]
      const ws1 = XLSX.utils.aoa_to_sheet(sheet1Data)
      ws1['!cols'] = [{ wch: 18 }, { wch: 14 }, { wch: 14 }, { wch: 28 }]

      try {
        const schoolNames = checkedSchools.map(s => s.name).join(',')
        ;(ws1 as unknown as { '!dataValidation'?: unknown[] })['!dataValidation'] = [
          { sqref: 'D2:D1001', type: 'list', formula1: `"${schoolNames}"`, allowBlank: false },
        ]
      } catch {
        // 下拉写入失败 → 忽略，老师可照 Sheet2 清单手填学校名（靠映射反查）
      }

      const sheet2Data = [
        SCHOOL_SHEET_HEADERS,
        ...checkedSchools.map(s => [s.name, s.id]),
      ]
      const ws2 = XLSX.utils.aoa_to_sheet(sheet2Data)
      ws2['!cols'] = [{ wch: 32 }, { wch: 38 }]

      const wb = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(wb, ws1, '教师名单')
      XLSX.utils.book_append_sheet(wb, ws2, '学校清单（请勿修改）')

      const regionName = regions.find(r => r.id === selectedRegionId)?.name ?? '区域'
      XLSX.writeFile(wb, `${regionName}_多校教师导入模板.xlsx`)
    } catch (e: unknown) {
      setError(e instanceof Error ? `生成模板失败：${e.message}` : '生成模板失败')
    } finally {
      setDownloading(false)
    }
  }, [checkedSchools, regions, selectedRegionId])

  const handleFileSelected = useCallback(async (file: File) => {
    setError('')
    setResult(null)
    setRows([])
    setFileName(file.name)
    setParsing(true)
    try {
      const XLSX = await import('xlsx')
      const buf = await file.arrayBuffer()
      const wb = XLSX.read(buf, { type: 'array' })

      let nameToId = nameToIdMapRef.current
      if (nameToId.size === 0) {
        const schoolSheetName = wb.SheetNames.find(n => n.includes('学校清单'))
        if (schoolSheetName) {
          const ws2 = wb.Sheets[schoolSheetName]
          const m2 = XLSX.utils.sheet_to_json<unknown[]>(ws2, { header: 1, defval: '' })
          const rebuilt = new Map<string, string>()
          for (let i = 1; i < m2.length; i++) {
            const name = String((m2[i]?.[0] ?? '')).trim()
            const id = String((m2[i]?.[1] ?? '')).trim()
            if (name && id) rebuilt.set(name, id)
          }
          nameToId = rebuilt
        }
      }

      const teacherSheetName =
        wb.SheetNames.find(n => !n.includes('学校清单')) ?? wb.SheetNames[0]
      if (!teacherSheetName) {
        setError('文件中没有任何工作表')
        return
      }
      const ws = wb.Sheets[teacherSheetName]
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
          schoolName: String((cols?.[3] ?? '')).trim(),
        }))
        .filter(r => r.username || r.displayName || r.password || r.schoolName)

      if (!raw.length) {
        setError('未读取到任何教师数据，请检查是否填在了前四列（用户名/姓名/密码/所属学校）')
        return
      }
      if (raw.length > MAX_ROWS) {
        setError(`单次最多导入 ${MAX_ROWS} 人，当前 ${raw.length} 人，请分批导入`)
        return
      }

      const preview: PreviewRow[] = raw.map((r, i) => {
        const index = i + 1
        const schoolId = r.schoolName ? (nameToId.get(r.schoolName) ?? '') : ''
        let localError = ''
        if (!r.username) localError = '用户名不能为空'
        else if (!r.displayName) localError = '教师姓名不能为空'
        else if (!r.password || r.password.length < 6) localError = '初始密码至少6位'
        else if (!r.schoolName) localError = '未填所属学校'
        else if (!schoolId) localError = `学校「${r.schoolName}」未匹配，请照"学校清单"填写`
        return {
          index,
          username: r.username,
          displayName: r.displayName,
          password: r.password,
          schoolName: r.schoolName,
          schoolId,
          localError,
        }
      })
      setRows(preview)
    } catch (e: unknown) {
      setError(e instanceof Error ? `解析失败：${e.message}` : '解析失败，请确认上传的是 Excel(.xlsx) 文件')
    } finally {
      setParsing(false)
    }
  }, [])

  const onInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0]
    if (f) handleFileSelected(f)
    e.target.value = ''
  }

  const localErrorCount = rows.filter(r => r.localError).length
  const canSubmit = rows.length > 0 && localErrorCount === 0 && !submitting && !parsing

  const handleSubmit = useCallback(async () => {
    if (rows.length === 0) { setError('请先上传并解析教师名单'); return }
    if (localErrorCount > 0) { setError(`有 ${localErrorCount} 行存在问题，请修正 Excel 后重新上传`); return }
    setError('')
    setResult(null)
    setSubmitting(true)
    try {
      const res = await batchCreateMultiSchoolUsers({
        role,
        users: rows.map(r => ({
          username: r.username,
          display_name: r.displayName,
          password: r.password,
          school_id: r.schoolId,
        })),
      })
      setResult(res)
      if (res.created_count > 0) {
        onImported(res.created_count)
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '跨校批量导入失败，请稍后重试')
    } finally {
      setSubmitting(false)
    }
  }, [rows, role, localErrorCount, onImported])

  const renamedList = result?.created.filter(c => c.renamed) ?? []
  const handleCopyRenamed = useCallback(async () => {
    if (renamedList.length === 0) return
    const idToNameLocal = new Map(schools.map(s => [s.id, s.name]))
    const lines = renamedList.map(c => {
      const sch = idToNameLocal.get(c.school_id) ?? c.school_id
      return `${c.display_name}（${sch}）：原填用户名 ${c.original_username} → 实际登录用户名 ${c.final_username}`
    })
    const text = '【因重名被系统自动改名的老师，请通知本人用新用户名登录】\n' + lines.join('\n')
    try {
      await navigator.clipboard.writeText(text)
      setCopyMsg('✓ 已复制改名清单，可粘贴发给学校')
      setTimeout(() => setCopyMsg(''), 2500)
    } catch {
      setCopyMsg('复制失败，请手动选择文字复制')
      setTimeout(() => setCopyMsg(''), 2500)
    }
  }, [renamedList, schools])

  const idToName = new Map(schools.map(s => [s.id, s.name]))

  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 10000,
        background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}
      onClick={e => { if (e.target === e.currentTarget && !submitting) onClose() }}>
      <div style={{ background: C.white, borderRadius: '20px', width: '820px', maxWidth: '95vw', maxHeight: '92vh', overflow: 'hidden', boxShadow: '0 20px 60px rgba(0,0,0,0.2)', display: 'flex', flexDirection: 'column' }}>

        <div style={{ padding: '20px 24px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0 }}>
          <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>🏫 跨区域多校批量导入教师</div>
          <button onClick={() => { if (!submitting) onClose() }} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>×</button>
        </div>

        <div style={{ padding: '24px', overflowY: 'auto', flex: 1 }}>

          <div style={{ background: C.primaryLight, borderRadius: '10px', padding: '14px 16px', marginBottom: '18px', fontSize: '13px', color: C.primary, lineHeight: 1.8 }}>
            <div style={{ fontWeight: 700, marginBottom: '4px' }}>适用：一次把一个区域下多所学校的老师，用一张表汇总导入。</div>
            ① 选区域 → 勾选学校 → 选角色 → 下载模板<br />
            ② 把模板发给各学校，老师填本校老师、在「所属学校」列选/填自己学校名<br />
            ③ 汇总后上传，系统逐校建好；重名会自动改名并给出清单
          </div>

          {error && (
            <div style={{ padding: '10px 14px', borderRadius: '8px', marginBottom: '16px', background: C.dangerLight, color: C.danger, fontSize: '13px', lineHeight: 1.6 }}>
              {error}
            </div>
          )}

          <div style={{ border: `1px solid ${C.border}`, borderRadius: '12px', padding: '16px', marginBottom: '18px' }}>
            <div style={{ fontSize: '13px', fontWeight: 700, color: C.text, marginBottom: '12px' }}>第一步 · 生成模板</div>

            <div style={{ marginBottom: '14px' }}>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                选择区域 <span style={{ color: C.danger }}>*</span>
              </label>
              <select
                value={selectedRegionId}
                onChange={e => handleRegionChange(e.target.value)}
                disabled={submitting}
                style={{ width: '100%', padding: '10px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: C.white }}>
                <option value="">请选择区域...</option>
                {regions.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
              </select>
            </div>

            {selectedRegionId && (
              <div style={{ marginBottom: '14px' }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '6px' }}>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>
                    勾选要导入的学校 <span style={{ color: C.danger }}>*</span>
                    {checkedSchoolIds.size > 0 && (
                      <span style={{ color: C.primary, marginLeft: '8px', fontSize: '12px' }}>已选 {checkedSchoolIds.size} 所</span>
                    )}
                  </label>
                  {schools.length > 0 && (
                    <button type="button" onClick={toggleSelectAll}
                      style={{ background: 'none', border: 'none', color: C.primary, fontSize: '12px', fontWeight: 600, cursor: 'pointer' }}>
                      {allChecked ? '取消全选' : '全选'}
                    </button>
                  )}
                </div>
                {loadingSchools ? (
                  <div style={{ fontSize: '13px', color: C.textMuted, padding: '10px 0' }}>加载学校中...</div>
                ) : (
                  <div style={{ border: `1px solid ${C.border}`, borderRadius: '10px', maxHeight: '180px', overflowY: 'auto', padding: '6px' }}>
                    {schools.map(s => {
                      const checked = checkedSchoolIds.has(s.id)
                      return (
                        <label key={s.id}
                          style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '7px 10px', borderRadius: '8px', cursor: 'pointer', background: checked ? C.primaryLight : 'transparent', fontSize: '13px', color: C.text }}>
                          <input type="checkbox" checked={checked} onChange={() => toggleSchool(s.id)} style={{ cursor: 'pointer' }} />
                          {s.name}
                        </label>
                      )
                    })}
                  </div>
                )}
              </div>
            )}

            <div style={{ marginBottom: '14px' }}>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
                这一批是哪类账号？<span style={{ color: C.danger }}>*</span>
              </label>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
                {ROLE_BATCHES.map(rb => {
                  const active = role === rb.value
                  return (
                    <button key={rb.value} type="button" onClick={() => { if (!submitting) setRole(rb.value) }} disabled={submitting}
                      style={{
                        textAlign: 'left', padding: '12px 14px', borderRadius: '12px',
                        border: active ? `2px solid ${C.primary}` : `1px solid ${C.border}`,
                        background: active ? C.primaryLight : C.white, cursor: submitting ? 'not-allowed' : 'pointer',
                      }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '3px' }}>
                        <span style={{ fontSize: '17px' }}>{rb.emoji}</span>
                        <span style={{ fontSize: '14px', fontWeight: 700, color: active ? C.primary : C.text }}>{rb.title}</span>
                        <span style={{ fontSize: '11px', color: C.textMuted, fontFamily: 'monospace' }}>{rb.subtitle}</span>
                        {active && <span style={{ marginLeft: 'auto', fontSize: '13px', color: C.primary, fontWeight: 700 }}>✓</span>}
                      </div>
                      <div style={{ fontSize: '11px', color: C.textSec }}>{rb.desc}</div>
                    </button>
                  )
                })}
              </div>
              <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '8px' }}>
                💡 这张表里 <b style={{ color: C.primary }}>全部</b> 老师都建为「{currentRole.title}」。要导另一类，请另出一张表。
              </div>
            </div>

            <button onClick={handleDownloadTemplate} disabled={downloading || checkedSchools.length === 0}
              style={{ padding: '10px 18px', borderRadius: '10px', border: `1px solid ${C.primary}`, background: checkedSchools.length === 0 ? C.bg : C.primaryLight, color: checkedSchools.length === 0 ? C.textMuted : C.primary, fontSize: '13px', fontWeight: 600, cursor: (downloading || checkedSchools.length === 0) ? 'not-allowed' : 'pointer' }}>
              {downloading ? '生成中...' : `⬇️ 下载 Excel 模板${checkedSchools.length > 0 ? `（含 ${checkedSchools.length} 所学校）` : ''}`}
            </button>
          </div>

          <div style={{ border: `1px solid ${C.border}`, borderRadius: '12px', padding: '16px', marginBottom: '4px' }}>
            <div style={{ fontSize: '13px', fontWeight: 700, color: C.text, marginBottom: '12px' }}>第二步 · 上传汇总名单</div>

            <div style={{ display: 'flex', gap: '10px', marginBottom: '6px', flexWrap: 'wrap', alignItems: 'center' }}>
              <button onClick={() => fileInputRef.current?.click()} disabled={parsing || submitting}
                style={{ padding: '10px 18px', borderRadius: '10px', border: 'none', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: (parsing || submitting) ? 'wait' : 'pointer' }}>
                {parsing ? '解析中...' : '📂 上传填好的 Excel'}
              </button>
              <input ref={fileInputRef} type="file"
                accept=".xlsx,.xls,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.ms-excel"
                style={{ display: 'none' }} onChange={onInputChange} />
              {fileName && <span style={{ fontSize: '12px', color: C.textSec }}>已选文件：{fileName}</span>}
            </div>
            {nameToIdMapRef.current.size === 0 && (
              <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px' }}>
                💡 若本次未在上方生成模板（如刷新过页面），上传时将自动读取表内「学校清单」反查学校，照样可用。
              </div>
            )}

            {rows.length > 0 && (
              <div style={{ marginTop: '14px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '8px', flexWrap: 'wrap' }}>
                  <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>预览（共 {rows.length} 行）</span>
                  <span style={{ fontSize: '12px', fontWeight: 600, color: C.primary, background: C.primaryLight, padding: '2px 10px', borderRadius: '999px' }}>
                    {currentRole.emoji} 本批角色：{currentRole.title}
                  </span>
                  {localErrorCount > 0
                    ? <span style={{ fontSize: '12px', color: C.danger }}>⚠️ {localErrorCount} 行有问题，需修正</span>
                    : <span style={{ fontSize: '12px', color: C.success }}>✓ 校验通过，可提交</span>}
                </div>
                <div style={{ border: `1px solid ${C.border}`, borderRadius: '10px', overflow: 'hidden' }}>
                  <div style={{ display: 'grid', gridTemplateColumns: '40px 1.2fr 1fr 0.8fr 1.4fr 1.4fr', padding: '10px 12px', background: C.bg, borderBottom: `1px solid ${C.border}`, fontSize: '12px', fontWeight: 600, color: C.textSec }}>
                    <span>#</span><span>用户名</span><span>姓名</span><span>密码</span><span>所属学校</span><span>校验</span>
                  </div>
                  <div style={{ maxHeight: '220px', overflowY: 'auto' }}>
                    {rows.map((r, idx) => {
                      const hasErr = !!r.localError
                      return (
                        <div key={r.index}
                          style={{ display: 'grid', gridTemplateColumns: '40px 1.2fr 1fr 0.8fr 1.4fr 1.4fr', padding: '8px 12px', fontSize: '12px', alignItems: 'center', borderBottom: idx < rows.length - 1 ? `1px solid ${C.border}` : 'none', background: hasErr ? C.dangerLight : 'transparent' }}>
                          <span style={{ color: C.textMuted }}>{r.index}</span>
                          <span style={{ color: C.text, fontWeight: 500 }}>{r.username || <em style={{ color: C.textMuted }}>空</em>}</span>
                          <span style={{ color: C.text }}>{r.displayName || <em style={{ color: C.textMuted }}>空</em>}</span>
                          <span style={{ color: C.textSec, fontFamily: 'monospace' }}>{r.password ? '•'.repeat(Math.min(r.password.length, 6)) : <em style={{ color: C.textMuted }}>空</em>}</span>
                          <span style={{ color: r.schoolId ? C.text : C.danger }}>{r.schoolName || <em style={{ color: C.textMuted }}>空</em>}</span>
                          <span style={{ color: hasErr ? C.danger : C.success, fontWeight: hasErr ? 600 : 400 }}>{hasErr ? r.localError : '✓'}</span>
                        </div>
                      )
                    })}
                  </div>
                </div>
              </div>
            )}
          </div>

          {result && (
            <div style={{ marginTop: '18px' }}>
              <div style={{ padding: '12px 16px', borderRadius: '10px', background: C.successLight, color: C.success, fontSize: '14px', fontWeight: 700, marginBottom: '12px' }}>
                导入完成：成功 {result.created_count} 人
                {result.failed_count > 0 && <span style={{ color: C.danger }}>，失败 {result.failed_count} 人</span>}
                （共 {result.total_count} 人）
              </div>

              {renamedList.length > 0 && (
                <div style={{ border: `1px solid ${C.primary}`, borderRadius: '10px', padding: '14px', marginBottom: '12px', background: C.primaryLight }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '8px', flexWrap: 'wrap', gap: '8px' }}>
                    <span style={{ fontSize: '13px', fontWeight: 700, color: C.primary }}>
                      ⚠️ {renamedList.length} 位老师因重名被自动改名，请通知本人
                    </span>
                    <button onClick={handleCopyRenamed}
                      style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.primary}`, background: C.white, color: C.primary, fontSize: '12px', fontWeight: 600, cursor: 'pointer' }}>
                      📋 一键复制清单
                    </button>
                  </div>
                  {copyMsg && <div style={{ fontSize: '12px', color: C.success, marginBottom: '6px' }}>{copyMsg}</div>}
                  <div style={{ maxHeight: '160px', overflowY: 'auto', fontSize: '12px', lineHeight: 1.9 }}>
                    {renamedList.map(c => {
                      const sch = idToName.get(c.school_id) ?? c.school_id
                      return (
                        <div key={c.index} style={{ color: C.text }}>
                          {c.display_name}（{sch}）：<span style={{ textDecoration: 'line-through', color: C.textMuted }}>{c.original_username}</span> → <b style={{ color: C.primary }}>{c.final_username}</b>
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}

              {result.failures.length > 0 && (
                <div style={{ border: `1px solid ${C.danger}`, borderRadius: '10px', padding: '14px', marginBottom: '12px' }}>
                  <div style={{ fontSize: '13px', fontWeight: 700, color: C.danger, marginBottom: '8px' }}>
                    以下 {result.failures.length} 行未能导入（其余已建好，可单独修正这些行再传一次）
                  </div>
                  <div style={{ maxHeight: '160px', overflowY: 'auto', fontSize: '12px', lineHeight: 1.9 }}>
                    {result.failures.map(f => {
                      const sch = idToName.get(f.school_id) ?? f.school_id ?? '—'
                      return (
                        <div key={f.index} style={{ color: C.text }}>
                          第 {f.index} 行 · {f.username}（{sch}）：<span style={{ color: C.danger }}>{f.reason}</span>
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        <div style={{ padding: '16px 24px', borderTop: `1px solid ${C.border}`, display: 'flex', gap: '10px', flexShrink: 0 }}>
          <button onClick={() => { if (!submitting) onClose() }}
            style={{ flex: 1, padding: '11px', borderRadius: '10px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '14px', color: C.textSec, cursor: submitting ? 'not-allowed' : 'pointer' }}>
            {result ? '关闭' : '取消'}
          </button>
          {!result && (
            <button onClick={handleSubmit} disabled={!canSubmit}
              style={{
                flex: 2, padding: '11px', borderRadius: '10px', border: 'none',
                background: canSubmit ? `linear-gradient(135deg,${C.primary},#7C3AED)` : C.textMuted,
                color: '#fff', fontSize: '14px', fontWeight: 600, cursor: canSubmit ? 'pointer' : 'not-allowed',
              }}>
              {submitting ? '导入中（请勿关闭）...' : rows.length > 0 ? `确认导入 ${rows.length} 位${currentRole.title}` : '请先上传名单'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
