/**
 * AppearancePanel.tsx — 外观设置面板（批次W2新建，W3加mode参数）
 * 聚合课件级外观设置：背景图库选择器 + 字体方案选择器。
 *   - Step3确认导航栏页：mode省略(both)，两个选择器纵向堆叠；
 *   - Step5工作台：背景/字体是两个独立Tab，分别以 mode="background"/"font" 单显
 *     （W3：避免外观Tab内下拉越来越长，也为将来图库翻页留出整Tab空间）。
 */
import BackgroundPicker from './BackgroundPicker'
import FontPicker from './FontPicker'

interface Props {
  coursewareId: string
  onSwapped: () => void
  disabled?: boolean
  cwTitle?: string
  cwSubject?: string
  cwGrade?: string
  /** both=背景+字体堆叠(默认, Step3用) / background=仅背景 / font=仅字体(Step5两个Tab分别用) */
  mode?: 'both' | 'background' | 'font'
  /** 当前选中的页码（传入后BackgroundPicker显示页级背景区块） */
  pageNum?: number
}

export default function AppearancePanel({ coursewareId, onSwapped, disabled = false, cwTitle = '', cwSubject = '', cwGrade = '', mode = 'both', pageNum }: Props) {
  return (
    <>
      {(mode === 'both' || mode === 'background') && (
        <BackgroundPicker coursewareId={coursewareId} onSwapped={onSwapped} disabled={disabled}
          cwTitle={cwTitle} cwSubject={cwSubject} cwGrade={cwGrade} pageNum={pageNum} />
      )}
      {(mode === 'both' || mode === 'font') && (
        <FontPicker coursewareId={coursewareId} onSwapped={onSwapped} disabled={disabled} />
      )}
    </>
  )
}
