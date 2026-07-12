/**
 * 我参与的集体备课区块 — JoinedCollabSection.tsx（阶段4新建）
 *
 * 解决"被邀请的参与者在自己界面找不到课件"的断点：
 *   参与者（非作者）被拉入集体备课后，这里列出他被邀请、且仍在进行中的课件，
 *   点击进入同一个课件工坊（/courseware/:id），即可一起微调。
 *
 * 自包含：进来调 listJoinedCollab()，列表为空则整块不渲染（不打扰没被邀请的人）。
 * 挂在课件列表页主列表上方。
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { listJoinedCollab } from '@/api/coursewares'
import type { JoinedCollabItem } from '@/api/coursewares'

export default function JoinedCollabSection() {
  const navigate = useNavigate()
  const [items, setItems] = useState<JoinedCollabItem[]>([])
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    listJoinedCollab()
      .then(r => setItems(r.coursewares || []))
      .catch(() => { /* 拉取失败静默,不显示区块 */ })
      .finally(() => setLoaded(true))
  }, [])

  // 未加载完 或 没有参与的集体备课 → 整块不渲染
  if (!loaded || items.length === 0) return null

  return (
    <div style={{ marginBottom: 24 }}>
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12,
        fontSize: 15, fontWeight: 600, color: '#10893e',
      }}>
        <span>👥 我参与的集体备课</span>
        <span style={{
          fontSize: 12, fontWeight: 500, color: '#10893e',
          background: '#e6f7ed', borderRadius: 10, padding: '1px 8px',
        }}>{items.length}</span>
      </div>

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
        {items.map(it => (
          <div
            key={it.id}
            onClick={() => navigate('/courseware/' + it.id)}
            style={{
              flex: '1 1 280px', minWidth: 260, maxWidth: 360,
              border: '1px solid #b7eb8f', borderRadius: 10, padding: '12px 14px',
              background: '#f6ffed', cursor: 'pointer', transition: 'box-shadow 150ms',
            }}
            onMouseEnter={e => (e.currentTarget.style.boxShadow = '0 2px 10px rgba(16,137,62,0.15)')}
            onMouseLeave={e => (e.currentTarget.style.boxShadow = 'none')}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 6 }}>
              <span style={{
                fontSize: 11, fontWeight: 600, color: '#10893e',
                background: '#d9f7be', borderRadius: 8, padding: '1px 7px',
              }}>🟢 集体备课中</span>
            </div>
            <div style={{ fontSize: 14, fontWeight: 600, color: '#333', marginBottom: 4 }}>
              {it.title}
            </div>
            <div style={{ fontSize: 12, color: '#888' }}>
              {it.subject} · {it.grade} · {it.page_count}页
            </div>
            <div style={{ fontSize: 12, color: '#888', marginTop: 4 }}>
              来自 <span style={{ color: '#10893e', fontWeight: 500 }}>{it.owner_name}</span> 的课件
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
