/**
 * CoursewareComicDirectCanvasQuestionPopover.tsx
 *
 * 问题卡画布内悬浮编辑面板：
 *   - 编辑题目；
 *   - 每行一个选项；
 *   - 选择正确答案；
 *   - 编辑答案解析；
 *   - 面板固定在画布内部，不再在图片下方展开表单。
 */

import type {
  CSSProperties,
} from 'react'

import type {
  CoursewareComicQuestionPopoverProps,
} from './CoursewareComicDirectCanvasSupport'

export function CoursewareComicQuestionPopover({
  element,
  disabled,
  onClose,
  onTextChange,
  onOptionsChange,
  onAnswerChange,
  onKeyDown,
}: CoursewareComicQuestionPopoverProps) {
  const question =
    element.question

  if (!question) {
    return null
  }

  return (
    <div
      style={questionPopoverStyle}
      onPointerDown={event =>
        event.stopPropagation()
      }
      onClick={event =>
        event.stopPropagation()
      }
    >
      <div style={popoverHeaderStyle}>
        <strong>
          编辑问题卡
        </strong>

        <button
          type="button"
          onClick={onClose}
          style={closeButtonStyle}
        >
          ×
        </button>
      </div>

      <label style={popoverLabelStyle}>
        题目

        <textarea
          value={question.question}
          onChange={event =>
            onTextChange(
              element.id,
              'question',
              event.target.value,
            )
          }
          onKeyDown={event => {
            onKeyDown(event)
          }}
          disabled={disabled}
          rows={2}
          style={popoverTextareaStyle}
        />
      </label>

      <label style={popoverLabelStyle}>
        选项（每行一个）

        <textarea
          value={
            question.options.join(
              '\n',
            )
          }
          onChange={event =>
            onOptionsChange(
              element.id,
              event.target.value,
            )
          }
          onKeyDown={event => {
            onKeyDown(event)
          }}
          disabled={disabled}
          rows={3}
          style={popoverTextareaStyle}
        />
      </label>

      <label style={popoverLabelStyle}>
        正确答案

        <select
          value={
            question.answer_index
          }
          onChange={event =>
            onAnswerChange(
              element.id,
              Number(
                event.target.value,
              ),
            )
          }
          disabled={
            disabled ||
            question.options.length ===
              0
          }
          style={popoverInputStyle}
        >
          {question.options.map(
            (
              option,
              index,
            ) => (
              <option
                key={
                  `${index}:${option}`
                }
                value={index}
              >
                {String.fromCharCode(
                  65 + index,
                )}
                . {option}
              </option>
            ),
          )}
        </select>
      </label>

      <label style={popoverLabelStyle}>
        答案解析

        <textarea
          value={
            question.explanation
          }
          onChange={event =>
            onTextChange(
              element.id,
              'explanation',
              event.target.value,
            )
          }
          onKeyDown={event => {
            onKeyDown(event)
          }}
          disabled={disabled}
          rows={2}
          style={popoverTextareaStyle}
        />
      </label>
    </div>
  )
}

const questionPopoverStyle:
  CSSProperties = {
    position: 'absolute',
    top: 10,
    right: 10,
    width:
      'min(310px,calc(100% - 20px))',
    maxHeight:
      'calc(100% - 20px)',
    overflow: 'auto',
    zIndex: 1000,
    padding: 10,
    borderRadius: 9,
    border:
      '1px solid #C4B5FD',
    background:
      'rgba(255,255,255,0.98)',
    boxShadow:
      '0 14px 34px rgba(15,23,42,0.28)',
    color: '#1F2937',
  }

const popoverHeaderStyle:
  CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    justifyContent:
      'space-between',
    marginBottom: 8,
    fontSize: 11,
  }

const closeButtonStyle:
  CSSProperties = {
    width: 25,
    height: 25,
    borderRadius: 999,
    border: 'none',
    background: '#F1F5F9',
    color: '#475569',
    fontSize: 16,
    cursor: 'pointer',
  }

const popoverLabelStyle:
  CSSProperties = {
    display: 'block',
    marginTop: 7,
    color: '#475569',
    fontSize: 9,
    fontWeight: 800,
  }

const popoverInputStyle:
  CSSProperties = {
    width: '100%',
    boxSizing: 'border-box',
    marginTop: 3,
    padding: '6px 7px',
    borderRadius: 6,
    border:
      '1px solid #CBD5E1',
    background: '#FFFFFF',
    color: '#1F2937',
    fontSize: 9,
    outline: 'none',
  }

const popoverTextareaStyle:
  CSSProperties = {
    ...popoverInputStyle,
    resize: 'vertical',
    lineHeight: 1.45,
  }
