/**
 * PageNumberCalibrationButton.tsx — 课件页码一键校准按钮。
 *
 * 老师确认胶片条当前从左到右顺序后，复用既有 reorderCWPages：
 *   - 后端按页面ID顺序原子重排为1..N；
 *   - 同步coursewares.page_count；
 *   - 刷新页面HTML导航栏里的当前页和总页数。
 *
 * 本组件不在前端直接改页面对象，成功后通知父级重新拉取正式页面列表。
 */
import { useState } from 'react'
import { reorderCWPages } from '@/api/coursewares'
import {
  buildPageNumberCalibrationPlan,
} from './pageNumberCalibration'

import type {
  PageNumberCalibrationItem,
} from './pageNumberCalibration'

interface Props {
  coursewareId: string
  pages: PageNumberCalibrationItem[]
  activePage: number
  disabled?: boolean
  onBusyChange: (busy: boolean) => void
  onSelectPage: (pageNumber: number) => void
  onPagesChanged?: () => void
  onMessage: (message: string, durationMs?: number) => void
}

export default function PageNumberCalibrationButton({
  coursewareId,
  pages,
  activePage,
  disabled = false,
  onBusyChange,
  onSelectPage,
  onPagesChanged,
  onMessage,
}: Props) {
  const [calibrating, setCalibrating] = useState(false)
  const blocked = disabled || calibrating || pages.length === 0

  const calibrate = async () => {
    if (blocked) return

    const result = buildPageNumberCalibrationPlan(
      pages,
      activePage,
    )

    if (!result.ok) {
      onMessage(result.message, 5000)
      return
    }

    const {
      pageIds,
      selectedPageNumber,
      orderText,
      total,
      alreadySequential,
    } = result.plan

    const stateNotice = alreadySequential
      ? '当前页号看起来已连续；本次仍会重新同步每页导航中的当前页和总页数。'
      : '当前胶片条视觉顺序将写入数据库，并重新编号为连续的1至N页。'

    const confirmed = window.confirm(
      `请确认课件从左到右的页面顺序：\n\n${orderText}`
      + `\n\n${stateNotice}`
      + `\n校准后总页数为 ${total}，是否继续？`,
    )

    if (!confirmed) return

    setCalibrating(true)
    onBusyChange(true)

    try {
      await reorderCWPages(
        coursewareId,
        pageIds,
      )

      // 当前选中页按其在视觉顺序中的新位置继续保持选中。
      onSelectPage(
        selectedPageNumber,
      )

      onPagesChanged?.()
      onMessage(
        `✅ 已按当前顺序校准 ${total} 页，导航中的当前页和总页数已同步`,
        5000,
      )
    } catch (error: unknown) {
      onMessage(
        '❌ 页码校准失败：'
        + (error instanceof Error ? error.message : '请刷新后重试'),
        5000,
      )
    } finally {
      setCalibrating(false)
      onBusyChange(false)
    }
  }

  return (
    <button
      type="button"
      onClick={() => void calibrate()}
      disabled={blocked}
      title="按当前胶片条顺序重新编号，并同步每页导航中的当前页和总页数"
      style={{
        padding: '6px 12px',
        borderRadius: 8,
        border: '1px solid #7C3AED',
        background: blocked
          ? '#F3F4F6'
          : 'rgba(124,58,237,0.06)',
        color: blocked
          ? '#9CA3AF'
          : '#6D28D9',
        fontSize: 13,
        fontWeight: 600,
        cursor: blocked
          ? 'default'
          : 'pointer',
      }}
    >
      {calibrating
        ? '⏳ 校准中…'
        : '🧭 校准页码'}
    </button>
  )
}

