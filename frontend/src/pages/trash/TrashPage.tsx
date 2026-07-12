import { useState, useEffect, useCallback } from 'react';
import { listTrash, restoreTrashItem, permanentDeleteTrashItem } from '@/api/trash'
import type { TrashItem } from '@/api/trash';

// ==================== 颜色常量 ====================
const C = {
  bg: '#F9FAFB', card: '#FFFFFF', border: '#E5E7EB',
  primary: '#3B82F6', primaryLight: '#EFF6FF',
  danger: '#EF4444', dangerLight: '#FEF2F2', dangerBorder: '#FECACA',
  success: '#10B981', successLight: '#ECFDF5',
  warn: '#F59E0B', warnLight: '#FFFBEB',
  text: '#111827', textSec: '#6B7280', textMuted: '#9CA3AF',
};

// ==================== 主页面 ====================
export default function TrashPage() {
  const [lpItems, setLpItems] = useState<TrashItem[]>([]);
  const [cwItems, setCwItems] = useState<TrashItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<'all' | 'lesson_plan' | 'courseware'>('all');
  const [toast, setToast] = useState('');
  // 确认弹窗状态
  const [confirmTarget, setConfirmTarget] = useState<TrashItem | null>(null);
  const [confirmAction, setConfirmAction] = useState<'restore' | 'delete' | null>(null);
  const [busy, setBusy] = useState(false);

  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      const data = await listTrash();
      setLpItems(data.lesson_plans || []);
      setCwItems(data.coursewares || []);
    } catch (e: any) {
      console.error('加载回收站失败', e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const showToast = (msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(''), 3000);
  };

  // 打开确认弹窗
  const openConfirm = (item: TrashItem, action: 'restore' | 'delete') => {
    setConfirmTarget(item);
    setConfirmAction(action);
  };

  // 执行确认操作
  const doConfirm = async () => {
    if (!confirmTarget || !confirmAction) return;
    setBusy(true);
    try {
      if (confirmAction === 'restore') {
        await restoreTrashItem(confirmTarget.id, confirmTarget.type);
        showToast(`✅ 已恢复「${confirmTarget.title}」`);
      } else {
        await permanentDeleteTrashItem(confirmTarget.id, confirmTarget.type);
        showToast(`🗑️ 已永久删除「${confirmTarget.title}」`);
      }
      setConfirmTarget(null);
      setConfirmAction(null);
      loadData();
    } catch (e: any) {
      showToast(`❌ 操作失败: ${e?.response?.data?.message || e.message}`);
    } finally {
      setBusy(false);
    }
  };

  // 按Tab过滤
  const visibleItems: TrashItem[] = tab === 'lesson_plan' ? lpItems
    : tab === 'courseware' ? cwItems
    : [...lpItems, ...cwItems].sort((a, b) => new Date(b.deleted_at).getTime() - new Date(a.deleted_at).getTime());

  const totalCount = lpItems.length + cwItems.length;

  return (
    <div style={{ padding: '24px 32px', maxWidth: 960, margin: '0 auto' }}>
      {/* 顶部标题 */}
      <h2 style={{ fontSize: 22, fontWeight: 700, color: C.text, marginBottom: 4 }}>
        🗑️ 回收站
      </h2>
      <p style={{ color: C.textSec, fontSize: 14, marginBottom: 20 }}>
        删除的教案和课件会在此保留30天，过期后将自动永久删除
      </p>

      {/* Tab栏 */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 20 }}>
        {([['all', '全部', totalCount], ['lesson_plan', '📝 教案', lpItems.length], ['courseware', '📊 课件', cwItems.length]] as const).map(([key, label, count]) => (
          <button key={key} onClick={() => setTab(key as any)} style={{
            padding: '6px 16px', borderRadius: 8, border: `1px solid ${tab === key ? C.primary : C.border}`,
            background: tab === key ? C.primaryLight : C.card, color: tab === key ? C.primary : C.textSec,
            fontWeight: tab === key ? 600 : 400, cursor: 'pointer', fontSize: 14,
          }}>
            {label} ({count})
          </button>
        ))}
      </div>

      {/* 列表 */}
      {loading ? (
        <p style={{ color: C.textMuted, textAlign: 'center', padding: 40 }}>加载中...</p>
      ) : visibleItems.length === 0 ? (
        <div style={{ textAlign: 'center', padding: 60, color: C.textMuted }}>
          <div style={{ fontSize: 48, marginBottom: 12 }}>🎉</div>
          <div>回收站为空</div>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {visibleItems.map(item => (
            <div key={item.id} style={{
              background: C.card, border: `1px solid ${C.border}`, borderRadius: 10,
              padding: '14px 18px', display: 'flex', alignItems: 'center', gap: 14,
            }}>
              {/* 类型图标 */}
              <span style={{ fontSize: 24 }}>{item.type === 'lesson_plan' ? '📝' : '📊'}</span>
              {/* 信息 */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontWeight: 600, color: C.text, fontSize: 15, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {item.title}
                </div>
                <div style={{ fontSize: 13, color: C.textSec, marginTop: 2 }}>
                  {item.subject && <span style={{ marginRight: 10 }}>{item.subject}</span>}
                  {item.grade && <span style={{ marginRight: 10 }}>{item.grade}</span>}
                  <span>删除于 {new Date(item.deleted_at).toLocaleDateString('zh-CN')}</span>
                </div>
              </div>
              {/* 剩余天数 */}
              <span style={{
                fontSize: 12, padding: '2px 8px', borderRadius: 10,
                background: item.days_left <= 3 ? C.dangerLight : item.days_left <= 7 ? C.warnLight : C.successLight,
                color: item.days_left <= 3 ? C.danger : item.days_left <= 7 ? C.warn : C.success,
                fontWeight: 500, whiteSpace: 'nowrap',
              }}>
                {item.days_left}天后过期
              </span>
              {/* 操作按钮 */}
              <button onClick={() => openConfirm(item, 'restore')} style={{
                padding: '5px 12px', borderRadius: 6, border: `1px solid ${C.primary}`,
                background: C.primaryLight, color: C.primary, cursor: 'pointer', fontSize: 13, fontWeight: 500,
              }}>♻️ 恢复</button>
              <button onClick={() => openConfirm(item, 'delete')} style={{
                padding: '5px 12px', borderRadius: 6, border: `1px solid ${C.dangerBorder}`,
                background: C.dangerLight, color: C.danger, cursor: 'pointer', fontSize: 13, fontWeight: 500,
              }}>🗑️ 永久删除</button>
            </div>
          ))}
        </div>
      )}

      {/* 确认弹窗 */}
      {confirmTarget && confirmAction && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', zIndex: 99999,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }} onClick={() => { if (!busy) { setConfirmTarget(null); setConfirmAction(null); } }}>
          <div style={{
            background: '#fff', borderRadius: 14, padding: '28px 32px', maxWidth: 420, width: '90%',
            boxShadow: '0 20px 60px rgba(0,0,0,0.15)',
          }} onClick={e => e.stopPropagation()}>
            <h3 style={{ margin: '0 0 12px', fontSize: 17, color: confirmAction === 'delete' ? C.danger : C.primary }}>
              {confirmAction === 'delete' ? '⚠️ 确认永久删除' : '♻️ 确认恢复'}
            </h3>
            <p style={{ color: C.textSec, fontSize: 14, margin: '0 0 8px', lineHeight: 1.6 }}>
              {confirmAction === 'delete'
                ? `永久删除「${confirmTarget.title}」后将无法恢复，所有关联数据也会被清除。`
                : `将「${confirmTarget.title}」恢复到${confirmTarget.type === 'lesson_plan' ? '我的教案' : '我的课件'}列表中。`
              }
            </p>
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 20 }}>
              <button onClick={() => { setConfirmTarget(null); setConfirmAction(null); }} disabled={busy}
                style={{ padding: '7px 18px', borderRadius: 8, border: `1px solid ${C.border}`, background: '#fff', color: C.textSec, cursor: 'pointer', fontSize: 14 }}>
                取消
              </button>
              <button onClick={doConfirm} disabled={busy}
                style={{
                  padding: '7px 18px', borderRadius: 8, border: 'none', fontSize: 14, fontWeight: 600, cursor: busy ? 'wait' : 'pointer',
                  background: confirmAction === 'delete' ? C.danger : C.primary, color: '#fff', opacity: busy ? 0.6 : 1,
                }}>
                {busy ? '处理中...' : confirmAction === 'delete' ? '确认永久删除' : '确认恢复'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Toast */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: 30, left: '50%', transform: 'translateX(-50%)',
          background: '#333', color: '#fff', padding: '10px 24px', borderRadius: 10,
          fontSize: 14, zIndex: 100000, boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
        }}>{toast}</div>
      )}
    </div>
  );
}
