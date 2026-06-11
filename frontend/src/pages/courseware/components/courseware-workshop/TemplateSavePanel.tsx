/**
 * TemplateSavePanel.tsx — 保存为我的模板面板（批次W2从主页面抽出，工作台「💾模板」Tab）
 * 把当前课件的风格与导航栏保存为个人模板，下次生成课件可复用。
 */
import { useState } from 'react'
import { saveAsMyTemplate } from '@/api/coursewares'
import { C } from './workshopConstants'

interface Props {
  coursewareId: string
}

export default function TemplateSavePanel({ coursewareId }: Props) {
  const [saveTplName, setSaveTplName] = useState('')
  const [savingTpl, setSavingTpl] = useState(false)

  return (
    <div style={{ marginTop: 16, padding: '16px', borderRadius: 10, border: `1px solid ${C.border}`, background: '#FAFAFA' }}>
      <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>💾 保存为我的模板（下次生成课件可复用当前风格和导航栏）</div>
      <div style={{ display: 'flex', gap: 10 }}>
        <input value={saveTplName} onChange={e => setSaveTplName(e.target.value)}
          placeholder="输入模板名称，如：我的品牌模板-蓝色版"
          style={{ flex: 1, padding: '10px 14px', borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 14, outline: 'none' }} />
        <button onClick={async () => {
          if (!coursewareId || !saveTplName.trim() || savingTpl) return
          setSavingTpl(true)
          try {
            const res = await saveAsMyTemplate(coursewareId, { name: saveTplName.trim() })
            alert(res.message || '模板保存成功！')
            setSaveTplName('')
          } catch (e) { alert('保存失败: ' + (e instanceof Error ? e.message : '未知错误')) }
          finally { setSavingTpl(false) }
        }} disabled={savingTpl || !saveTplName.trim()}
          style={{ padding: '10px 20px', borderRadius: 8, border: 'none', background: saveTplName.trim() && !savingTpl ? '#059669' : '#E5E7EB', color: saveTplName.trim() && !savingTpl ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: saveTplName.trim() && !savingTpl ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
          {savingTpl ? '⏳ 保存中...' : '💾 保存模板'}
        </button>
      </div>
    </div>
  )
}
