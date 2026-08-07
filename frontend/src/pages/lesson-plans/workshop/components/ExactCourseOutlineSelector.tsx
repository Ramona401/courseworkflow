/**
 * ExactCourseOutlineSelector — 自动匹配与手动回退共用的课程大纲选择器
 *
 * 交互规则：
 * 1. 学科或年级变化后，先请求exact模式候选；
 * 2. exact恰好一条时确定性自动绑定；
 * 3. exact为零条或多条时不进行猜测，展示手动选择入口；
 * 4. 教师点击手动选择后，请求manual模式候选；
 * 5. manual候选可以包含覆盖当前年级的学段大纲；
 * 6. 正式值始终是唯一course_outline_id；
 * 7. 教师始终可以明确选择“不关联课程大纲”；
 * 8. 加载期间通知父页面禁止开始请求，防止提交旧ID；
 * 9. 请求序号用于丢弃学科或年级切换后的过期响应。
 */

import {
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import {
  getExactCourseOutlineCandidates,
  getManualCourseOutlineCandidates,
  publisherLabel,
  schoolSystemLabel,
} from '@/api/course-outlines'
import type {
  ExactCourseOutlineCandidate,
} from '@/api/course-outlines'
import { C } from './workshopConstants'

type SelectorStatus =
  | 'idle'
  | 'auto'
  | 'manual_required'
  | 'manual'

type RequestPhase =
  | 'exact'
  | 'manual'

interface ExactCourseOutlineSelectorProps {
  /** 当前学科。空值时不请求。 */
  subject: string

  /** 当前具体年级或学习层级。空值时不请求。 */
  grade: string

  /** 当前选择的唯一大纲ID；null表示明确不关联。 */
  value: string | null

  /** 选择变化回调。 */
  onChange: (value: string | null) => void

  /** 父页面其它业务是否要求禁用。 */
  disabled?: boolean

  /** 选择器标题。 */
  label?: string

  /** 候选加载状态回调，父页面据此阻止提交旧ID。 */
  onLoadingChange?: (loading: boolean) => void
}

/** 从统一Axios错误或普通Error中提取教师可读信息。 */
function courseOutlineErrorMessage(
  error: unknown,
): string {
  if (
    error &&
    typeof error === 'object'
  ) {
    const candidate = error as {
      message?: unknown
      response?: {
        data?: {
          message?: unknown
          error?: unknown
        }
      }
    }

    const responseMessage =
      candidate.response?.data?.message

    if (
      typeof responseMessage === 'string' &&
      responseMessage.trim()
    ) {
      return responseMessage.trim()
    }

    const responseError =
      candidate.response?.data?.error

    if (
      typeof responseError === 'string' &&
      responseError.trim()
    ) {
      return responseError.trim()
    }

    if (
      typeof candidate.message === 'string' &&
      candidate.message.trim()
    ) {
      return candidate.message.trim()
    }
  }

  return '课程大纲候选加载失败，请稍后重试'
}

/** 按唯一ID稳定去重，不合并不同大纲。 */
function uniqueCourseOutlineCandidates(
  candidates: ExactCourseOutlineCandidate[],
): ExactCourseOutlineCandidate[] {
  const seen = new Set<string>()

  return candidates.filter(candidate => {
    const id = candidate.id?.trim()

    if (!id || seen.has(id)) {
      return false
    }

    seen.add(id)
    return true
  })
}

/** 构造完整候选文案，并显式展示年级或学段。 */
function candidateLabel(
  candidate: ExactCourseOutlineCandidate,
): string {
  const parts = [
    candidate.title,
    candidate.grade,
    publisherLabel(candidate.publisher),
    candidate.volume,
    schoolSystemLabel(candidate.school_system),
    candidate.scope_name,
  ]
    .map(item => item?.trim())
    .filter(Boolean)

  return parts.join(' · ')
}

export default function ExactCourseOutlineSelector({
  subject,
  grade,
  value,
  onChange,
  disabled = false,
  label = '📚 课程大纲（选填）',
  onLoadingChange,
}: ExactCourseOutlineSelectorProps) {
  const [candidates, setCandidates] =
    useState<ExactCourseOutlineCandidate[]>([])

  const [status, setStatus] =
    useState<SelectorStatus>('idle')

  const [loading, setLoading] =
    useState(false)

  const [loadingPhase, setLoadingPhase] =
    useState<RequestPhase>('exact')

  const [error, setError] =
    useState('')

  const [errorPhase, setErrorPhase] =
    useState<RequestPhase>('exact')

  const [exactCandidateCount, setExactCandidateCount] =
    useState(0)

  const [reloadVersion, setReloadVersion] =
    useState(0)

  /**
   * 使用ref镜像当前值和回调，避免父页面重新创建函数导致候选请求重跑。
   */
  const valueRef = useRef(value)
  const onChangeRef = useRef(onChange)
  const onLoadingChangeRef =
    useRef(onLoadingChange)

  /**
   * 每个请求获得递增序号。
   * 学科、年级或请求模式变化时，新请求会使旧响应自动失效。
   */
  const requestSequenceRef = useRef(0)

  /**
   * 用于区分首次挂载和真实的学科、年级变化。
   * 首次挂载可复核并保留已有草稿选择；
   * 真实切换时必须立即清理旧ID。
   */
  const lastQueryKeyRef = useRef('')

  useEffect(() => {
    valueRef.current = value
  }, [value])

  useEffect(() => {
    onChangeRef.current = onChange
  }, [onChange])

  useEffect(() => {
    onLoadingChangeRef.current =
      onLoadingChange
  }, [onLoadingChange])

  const normalizedSubject =
    subject.trim()

  const normalizedGrade =
    grade.trim()

  /** 同步更新ref和父页面正式值。 */
  const emitChange = (
    nextValue: string | null,
  ) => {
    valueRef.current = nextValue
    onChangeRef.current(nextValue)
  }

  /** 统一进入加载态。 */
  const beginLoading = (
    phase: RequestPhase,
  ) => {
    setLoading(true)
    setLoadingPhase(phase)
    setError('')
    setErrorPhase(phase)
    onLoadingChangeRef.current?.(true)
  }

  /** 仅由仍然有效的请求结束加载态。 */
  const finishLoading = (
    requestSequence: number,
  ) => {
    if (
      requestSequenceRef.current !==
      requestSequence
    ) {
      return
    }

    setLoading(false)
    onLoadingChangeRef.current?.(false)
  }

  /**
   * exact模式自动解析。
   *
   * 首次挂载若已有选择：
   *   - 先验证它是否仍在exact候选；
   *   - 若不在，再用manual候选复核；
   *   - 两种候选都不存在时才清空。
   *
   * 学科或年级真实变化时，不继承旧选择。
   */
  useEffect(() => {
    const requestSequence =
      ++requestSequenceRef.current

    const queryKey =
      `${normalizedSubject}\u0000${normalizedGrade}`

    const previousQueryKey =
      lastQueryKeyRef.current

    const queryChanged =
      previousQueryKey !== '' &&
      previousQueryKey !== queryKey

    lastQueryKeyRef.current = queryKey

    if (
      !normalizedSubject ||
      !normalizedGrade
    ) {
      setCandidates([])
      setStatus('idle')
      setExactCandidateCount(0)
      setError('')
      setLoading(false)
      onLoadingChangeRef.current?.(false)

      if (valueRef.current !== null) {
        emitChange(null)
      }

      return () => {
        if (
          requestSequenceRef.current ===
          requestSequence
        ) {
          requestSequenceRef.current++
        }
      }
    }

    if (
      queryChanged &&
      valueRef.current !== null
    ) {
      emitChange(null)
    }

    const valueToValidate =
      queryChanged
        ? null
        : valueRef.current

    setCandidates([])
    setStatus('idle')
    setExactCandidateCount(0)
    beginLoading('exact')

    void (async () => {
      try {
        const exactCandidates =
          uniqueCourseOutlineCandidates(
            await getExactCourseOutlineCandidates(
              normalizedSubject,
              normalizedGrade,
            ),
          )

        if (
          requestSequenceRef.current !==
          requestSequence
        ) {
          return
        }

        setExactCandidateCount(
          exactCandidates.length,
        )

        if (valueToValidate) {
          const exactSelection =
            exactCandidates.find(
              item =>
                item.id === valueToValidate,
            )

          if (exactSelection) {
            setCandidates(exactCandidates)

            setStatus(
              exactCandidates.length === 1
                ? 'auto'
                : 'manual',
            )

            return
          }

          /**
           * 已有值可能是此前合法选择的学段大纲。
           * 首次恢复时需要用manual模式重新授权，不能因为它不在exact列表
           * 就直接清除。
           */
          const manualCandidates =
            uniqueCourseOutlineCandidates(
              await getManualCourseOutlineCandidates(
                normalizedSubject,
                normalizedGrade,
              ),
            )

          if (
            requestSequenceRef.current !==
            requestSequence
          ) {
            return
          }

          const manualSelection =
            manualCandidates.find(
              item =>
                item.id === valueToValidate,
            )

          if (manualSelection) {
            setCandidates(manualCandidates)
            setStatus('manual')
            return
          }

          emitChange(null)
        }

        if (exactCandidates.length === 1) {
          const onlyCandidate =
            exactCandidates[0]

          setCandidates(exactCandidates)
          setStatus('auto')
          emitChange(onlyCandidate.id)
          return
        }

        /**
         * 零条表示没有具体年级精确候选；
         * 多条表示无法安全确定出版社、册次或归属。
         * 两种情况都不自动猜测，交由老师手动选择。
         */
        setCandidates([])
        setStatus('manual_required')

        if (valueRef.current !== null) {
          emitChange(null)
        }
      } catch (requestError) {
        if (
          requestSequenceRef.current !==
          requestSequence
        ) {
          return
        }

        setCandidates([])
        setStatus('idle')
        setErrorPhase('exact')
        setError(
          courseOutlineErrorMessage(
            requestError,
          ),
        )

        if (valueRef.current !== null) {
          emitChange(null)
        }
      } finally {
        finishLoading(requestSequence)
      }
    })()

    return () => {
      if (
        requestSequenceRef.current ===
        requestSequence
      ) {
        requestSequenceRef.current++
      }
    }
  }, [
    normalizedSubject,
    normalizedGrade,
    reloadVersion,
  ])

  /** 教师主动进入manual模式并加载年级或学段相交候选。 */
  const loadManualCandidates = async () => {
    if (
      !normalizedSubject ||
      !normalizedGrade ||
      loading
    ) {
      return
    }

    const requestSequence =
      ++requestSequenceRef.current

    beginLoading('manual')

    try {
      const manualCandidates =
        uniqueCourseOutlineCandidates(
          await getManualCourseOutlineCandidates(
            normalizedSubject,
            normalizedGrade,
          ),
        )

      if (
        requestSequenceRef.current !==
        requestSequence
      ) {
        return
      }

      setCandidates(manualCandidates)
      setStatus('manual')

      const currentValue =
        valueRef.current

      if (
        currentValue !== null &&
        !manualCandidates.some(
          item => item.id === currentValue,
        )
      ) {
        emitChange(null)
      }
    } catch (requestError) {
      if (
        requestSequenceRef.current !==
        requestSequence
      ) {
        return
      }

      setErrorPhase('manual')
      setError(
        courseOutlineErrorMessage(
          requestError,
        ),
      )
    } finally {
      finishLoading(requestSequence)
    }
  }

  const uniqueCandidates = useMemo(
    () =>
      uniqueCourseOutlineCandidates(
        candidates,
      ),
    [candidates],
  )

  const selectVisible =
    status === 'auto' ||
    status === 'manual'

  const selectDisabled =
    disabled ||
    loading ||
    !normalizedSubject ||
    !normalizedGrade

  const manualReason =
    exactCandidateCount === 0
      ? '没有找到与当前具体年级完全相等的课程大纲。'
      : `自动匹配到${exactCandidateCount}份具体年级大纲，系统不能安全替您决定出版社、册次或归属。`

  return (
    <div style={{
      textAlign: 'left',
    }}>
      <label style={{
        display: 'block',
        fontSize: '12px',
        fontWeight: 600,
        color: C.textSec,
        marginBottom: '6px',
      }}>
        {label}
      </label>

      {selectVisible && (
        <select
          value={value ?? ''}
          disabled={selectDisabled}
          onChange={event => {
            const nextValue =
              event.target.value.trim()

            emitChange(
              nextValue || null,
            )
          }}
          style={{
            width: '100%',
            padding: '11px 14px',
            borderRadius: '12px',
            border: `1.5px solid ${
              value ? '#8B5CF6' : C.border
            }`,
            fontSize: '14px',
            background:
              selectDisabled
                ? '#F3F4F6'
                : C.card,
            color: C.text,
            cursor:
              selectDisabled
                ? 'not-allowed'
                : 'pointer',
          }}
        >
          <option value="">
            不关联课程大纲
          </option>

          {uniqueCandidates.map(candidate => (
            <option
              key={candidate.id}
              value={candidate.id}
            >
              {candidateLabel(candidate)}
            </option>
          ))}
        </select>
      )}

      {!loading &&
       !error &&
       status === 'auto' && (
        <div style={{
          marginTop: '7px',
          padding: '8px 10px',
          borderRadius: '8px',
          background: '#F0FDF4',
          border: '1px solid #BBF7D0',
          color: '#166534',
          fontSize: '11px',
          lineHeight: 1.65,
        }}>
          ✓ 已自动匹配唯一的具体年级课程大纲。
          您仍可选择“不关联课程大纲”。
        </div>
      )}

      {!loading &&
       !error &&
       status === 'manual_required' && (
        <div style={{
          padding: '10px 12px',
          borderRadius: '10px',
          background: '#FFF7ED',
          border: '1px solid #FED7AA',
          color: '#9A3412',
          fontSize: '11px',
          lineHeight: 1.65,
        }}>
          <div>
            {manualReason}
          </div>

          <div style={{
            marginTop: '4px',
            color: '#7C2D12',
          }}>
            您可以手动查看本人有权绑定的课程大纲，
            其中可以包含覆盖当前年级的学段大纲。
          </div>

          <button
            type="button"
            disabled={disabled}
            onClick={() => {
              void loadManualCandidates()
            }}
            style={{
              marginTop: '9px',
              padding: '6px 12px',
              borderRadius: '8px',
              border: '1px solid #FDBA74',
              background: '#FFFFFF',
              color: '#C2410C',
              fontSize: '12px',
              fontWeight: 600,
              cursor:
                disabled
                  ? 'not-allowed'
                  : 'pointer',
            }}
          >
            手动选择可绑定课程大纲
          </button>

          <span style={{
            marginLeft: '9px',
            color: C.textMuted,
          }}>
            也可保持不关联
          </span>
        </div>
      )}

      {!loading &&
       !error &&
       status === 'manual' &&
       uniqueCandidates.length > 0 && (
        <div style={{
          marginTop: '7px',
          fontSize: '11px',
          color: C.textMuted,
          lineHeight: 1.65,
        }}>
          已进入手动选择。
          候选均经过教育域、资源范围、学科及年级或学段匹配校验。
        </div>
      )}

      {!loading &&
       !error &&
       status === 'manual' &&
       uniqueCandidates.length === 0 && (
        <div style={{
          marginTop: '7px',
          padding: '8px 10px',
          borderRadius: '8px',
          background: '#F8FAFC',
          border: `1px solid ${C.border}`,
          color: C.textMuted,
          fontSize: '11px',
          lineHeight: 1.65,
        }}>
          当前没有本人有权绑定且与当前年级或学段匹配的课程大纲，
          本次可保持不关联。
        </div>
      )}

      {loading && (
        <div style={{
          marginTop: '6px',
          fontSize: '11px',
          color: C.textMuted,
          lineHeight: 1.6,
        }}>
          {loadingPhase === 'exact'
            ? '正在自动匹配当前具体年级课程大纲…'
            : '正在加载本人可手动绑定的年级或学段课程大纲…'}
        </div>
      )}

      {!loading && error && (
        <div
          role="alert"
          aria-live="polite"
          style={{
            marginTop: '7px',
            padding: '8px 10px',
            borderRadius: '8px',
            background: '#FEF2F2',
            border: '1px solid #FECACA',
            color: '#B91C1C',
            fontSize: '11px',
            lineHeight: 1.6,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '10px',
          }}
        >
          <span>
            ⚠️ {error}
          </span>

          <button
            type="button"
            onClick={() => {
              if (errorPhase === 'manual') {
                void loadManualCandidates()
                return
              }

              setReloadVersion(
                current => current + 1,
              )
            }}
            style={{
              flexShrink: 0,
              padding: '3px 9px',
              borderRadius: '6px',
              border: '1px solid #FCA5A5',
              background: '#FFFFFF',
              color: '#B91C1C',
              fontSize: '11px',
              cursor: 'pointer',
            }}
          >
            重试
          </button>
        </div>
      )}
    </div>
  )
}
