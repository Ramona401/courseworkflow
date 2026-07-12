/**
 * GeographyLabModal.tsx — 地理互动实验室编辑器弹窗
 *
 * 功能：
 *   - 地理模板分组选择；
 *   - 数值、开关、选项三类初始参数；
 *   - AI新建地理互动组件；
 *   - 基于现有模板AI改编与继续追改；
 *   - 独立iframe预览；
 *   - 统一底部课堂控制条；
 *   - 融入当前课件页。
 */

import {
  useEffect,
  useMemo,
  useState,
} from 'react'

import { C } from './workshopConstants'

import {
  GEOGRAPHY_LAB_DEFAULT_SIZE_INDEX,
  GEOGRAPHY_LAB_SIZE_PRESETS,
  buildDefaultGeographyLabParams,
  buildGeographyLabLayoutOverride,
  buildGeographyLabRefineInstruction,
} from './geographyLabUtils'

import type {
  GeographyLabParamValue,
  GeographyLabTemplate,
} from './geographyLabUtils'

import {
  GEOGRAPHY_LAB_TEMPLATES,
  getGeographyLabGroups,
} from './geographyLabTemplates'

import ExperimentAIPanel from './ExperimentAIPanel'

interface Props {
  pageNum: number
  onInsert: (instruction: string) => void
  onClose: () => void
  inserting?: boolean
}

const MODAL_CSS = [
  '.geolab-card{',
  'transition:border-color .15s,box-shadow .15s,transform .15s;',
  '}',

  '.geolab-card:hover{',
  'border-color:#2DD4BF!important;',
  'box-shadow:0 8px 24px rgba(15,118,110,.16);',
  'transform:translateY(-1px);',
  '}',

  '.geolab-ai-btn{',
  'opacity:0;',
  'transition:opacity .15s;',
  '}',

  '.geolab-card:hover .geolab-ai-btn{',
  'opacity:1;',
  '}',

  '.geolab-range{',
  '-webkit-appearance:none;',
  'appearance:none;',
  'width:100%;',
  'height:6px;',
  'border-radius:3px;',
  'background:#CCFBF1;',
  'outline:none;',
  'cursor:pointer;',
  '}',

  '.geolab-range::-webkit-slider-thumb{',
  '-webkit-appearance:none;',
  'appearance:none;',
  'width:18px;',
  'height:18px;',
  'border-radius:50%;',
  'background:linear-gradient(135deg,#2DD4BF,#0F766E);',
  'border:2.5px solid #fff;',
  'box-shadow:0 1px 4px rgba(15,118,110,.45);',
  'cursor:pointer;',
  '}',

  '.geolab-range::-moz-range-thumb{',
  'width:18px;',
  'height:18px;',
  'border-radius:50%;',
  'background:#0F766E;',
  'border:2.5px solid #fff;',
  'box-shadow:0 1px 4px rgba(15,118,110,.45);',
  'cursor:pointer;',
  '}',
].join('\n')

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function replaceRootId(
  html: string,
  rootId: string,
): string {
  return html
    .split('__ROOT_ID__')
    .join(rootId)
}

function Toggle({
  checked,
  onChange,
}: {
  checked: boolean
  onChange: (value: boolean) => void
}) {
  return (
    <div
      onClick={() => onChange(!checked)}
      style={{
        width: 40,
        height: 22,
        borderRadius: 11,
        flexShrink: 0,
        cursor: 'pointer',
        background: checked
          ? 'linear-gradient(135deg,#2DD4BF,#0F766E)'
          : '#D1D5DB',
        position: 'relative',
        transition: 'background 0.18s',
      }}
    >
      <div
        style={{
          position: 'absolute',
          top: 2,
          left: checked ? 20 : 2,
          width: 18,
          height: 18,
          borderRadius: '50%',
          background: '#fff',
          boxShadow: '0 1px 3px rgba(0,0,0,0.28)',
          transition: 'left 0.18s',
        }}
      />
    </div>
  )
}

export default function GeographyLabModal({
  pageNum,
  onInsert,
  onClose,
  inserting,
}: Props) {
  const firstTemplate = GEOGRAPHY_LAB_TEMPLATES[0]

  const [activeTplId, setActiveTplId] = useState(
    firstTemplate.id,
  )

  const activeTpl =
    GEOGRAPHY_LAB_TEMPLATES.find(
      template => template.id === activeTplId,
    ) || firstTemplate

  const [params, setParams] = useState<
    Record<string, GeographyLabParamValue>
  >(
    buildDefaultGeographyLabParams(activeTpl),
  )

  const [previewParams, setPreviewParams] = useState<
    Record<string, GeographyLabParamValue>
  >(params)

  const [sizeIdx, setSizeIdx] = useState(
    GEOGRAPHY_LAB_DEFAULT_SIZE_INDEX,
  )

  const [caption, setCaption] = useState('')
  const [positionHint, setPositionHint] = useState('')

  const [aiMode, setAiMode] = useState<
    'adapt' | 'create' | null
  >(null)

  const [aiCode, setAiCode] = useState('')
  const [aiBaseCode, setAiBaseCode] = useState('')

  const size =
    GEOGRAPHY_LAB_SIZE_PRESETS[sizeIdx] ||
    GEOGRAPHY_LAB_SIZE_PRESETS[
      GEOGRAPHY_LAB_DEFAULT_SIZE_INDEX
    ]

  const isAI = aiMode !== null

  useEffect(() => {
    const timer = window.setTimeout(
      () => setPreviewParams(params),
      220,
    )

    return () => {
      window.clearTimeout(timer)
    }
  }, [params])

  const previewDoc = useMemo(() => {
    const rootId = 'geography-lab-preview-root'
    let html = ''

    if (isAI) {
      if (aiCode.trim()) {
        html =
          replaceRootId(aiCode, rootId) +
          buildGeographyLabLayoutOverride(rootId)
      } else if (aiMode === 'adapt') {
        html =
          activeTpl.buildHTML(
            previewParams,
            rootId,
          ) +
          buildGeographyLabLayoutOverride(rootId)
      } else {
        html =
          '<div style="'
          + 'width:100%;'
          + 'height:100%;'
          + 'box-sizing:border-box;'
          + 'border:1px dashed #99F6E4;'
          + 'border-radius:16px;'
          + 'background:linear-gradient(135deg,#F0FDFA,#EFF6FF);'
          + 'display:flex;'
          + 'flex-direction:column;'
          + 'align-items:center;'
          + 'justify-content:center;'
          + 'font-family:-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;'
          + 'color:#115E59;'
          + '">'
          + '<div style="font-size:42px;margin-bottom:12px;">🌍</div>'
          + '<div style="font-size:18px;font-weight:850;">'
          + 'AI新建地理互动探究'
          + '</div>'
          + '<div style="font-size:13px;color:#64748B;margin-top:8px;">'
          + '在右侧描述地理探究，或上传地图、图表和教材图片'
          + '</div>'
          + '</div>'
      }
    } else {
      html =
        activeTpl.buildHTML(
          previewParams,
          rootId,
        ) +
        buildGeographyLabLayoutOverride(rootId)
    }

    const captionBlock = caption.trim()
      ? '<div style="'
        + 'text-align:center;'
        + 'font-size:14px;'
        + 'color:#64748B;'
        + 'margin-top:8px;'
        + 'font-style:italic;'
        + '">'
        + escapeHtml(caption.trim())
        + '</div>'
      : ''

    return '<!doctype html>'
      + '<html>'
      + '<head>'
      + '<meta charset="utf-8">'
      + '<style>'
      + 'html,body{'
      + 'margin:0;'
      + 'padding:0;'
      + 'background:transparent;'
      + 'overflow:hidden;'
      + '}'
      + 'body{'
      + 'display:flex;'
      + 'flex-direction:column;'
      + 'align-items:center;'
      + 'justify-content:flex-start;'
      + 'font-family:-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;'
      + '}'
      + '</style>'
      + '</head>'
      + '<body>'
      + '<div style="width:'
      + size.width
      + 'px;height:'
      + size.height
      + 'px;">'
      + html
      + '</div>'
      + captionBlock
      + '</body>'
      + '</html>'
  }, [
    isAI,
    aiMode,
    aiCode,
    activeTpl,
    previewParams,
    size.width,
    size.height,
    caption,
  ])

  const handlePickTemplate = (
    template: GeographyLabTemplate,
  ) => {
    const nextParams =
      buildDefaultGeographyLabParams(template)

    setActiveTplId(template.id)
    setParams(nextParams)
    setPreviewParams(nextParams)

    setAiMode(null)
    setAiCode('')
    setAiBaseCode('')
  }

  const enterAdaptMode = (
    template: GeographyLabTemplate,
  ) => {
    const nextParams =
      template.id === activeTplId
        ? params
        : buildDefaultGeographyLabParams(template)

    setActiveTplId(template.id)
    setParams(nextParams)
    setPreviewParams(nextParams)

    setAiBaseCode(
      template.buildHTML(
        nextParams,
        '__ROOT_ID__',
      ),
    )

    setAiCode('')
    setAiMode('adapt')
  }

  const enterCreateMode = () => {
    setAiBaseCode('')
    setAiCode('')
    setAiMode('create')
  }

  const exitAIMode = () => {
    setAiMode(null)
    setAiCode('')
    setAiBaseCode('')
  }

  const setParam = (
    key: string,
    value: GeographyLabParamValue,
  ) => {
    setParams(previous => ({
      ...previous,
      [key]: value,
    }))
  }

  const handleInsert = () => {
    if (inserting) return

    let templateForInsert:
      GeographyLabTemplate = activeTpl

    let paramsForInsert:
      Record<string, GeographyLabParamValue> = params

    if (isAI) {
      if (!aiCode.trim()) return

      const frozenHTML = aiCode

      templateForInsert = {
        id: 'ai-geography-lab',
        group: '✨ AI定制',
        name:
          aiMode === 'adapt'
            ? activeTpl.name + '·AI改编'
            : 'AI自定义地理互动探究',
        emoji: '✨',
        desc: 'AI定制的地理互动探究组件',
        params: [],
        buildHTML: (_params, rootId) =>
          replaceRootId(
            frozenHTML,
            rootId,
          ),
      }

      paramsForInsert = {}
    }

    const instruction =
      buildGeographyLabRefineInstruction({
        lab: {
          template: templateForInsert,
          params: paramsForInsert,
          width: size.width,
          height: size.height,
          caption:
            caption.trim() || undefined,
        },
        positionHint:
          positionHint.trim() || undefined,
      })

    onInsert(instruction)
  }

  const insertDisabled =
    Boolean(inserting) ||
    (isAI && !aiCode.trim())

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(8,32,34,0.64)',
        backdropFilter: 'blur(5px)',
        zIndex: 99993,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
      onClick={() => {
        if (!inserting) onClose()
      }}
    >
      <style>{MODAL_CSS}</style>

      <div
        style={{
          width: 'min(1520px,98vw)',
          height: 'min(900px,96vh)',
          background: '#fff',
          borderRadius: 24,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
          boxShadow:
            '0 34px 88px rgba(0,0,0,0.38)',
        }}
        onClick={event => event.stopPropagation()}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 14,
            padding: '16px 24px',
            background:
              'linear-gradient(135deg,#2DD4BF 0%,#0F766E 52%,#164E63 100%)',
            flexShrink: 0,
          }}
        >
          <span
            style={{
              width: 48,
              height: 48,
              borderRadius: 16,
              background: 'rgba(255,255,255,0.18)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 26,
              flexShrink: 0,
            }}
          >
            🌍
          </span>

          <div style={{ minWidth: 0 }}>
            <div
              style={{
                fontSize: 19,
                fontWeight: 850,
                color: '#fff',
                letterSpacing: 0.3,
              }}
            >
              地理互动实验室
            </div>

            <div
              style={{
                fontSize: 12.5,
                color: 'rgba(255,255,255,0.82)',
                marginTop: 2,
              }}
            >
              模板点选 · AI改编/新建 ·
              底部课堂控制条 · 纯HTML离线运行
            </div>
          </div>

          {isAI && (
            <span
              style={{
                padding: '6px 15px',
                borderRadius: 999,
                background: 'rgba(255,255,255,0.24)',
                border:
                  '1px solid rgba(255,255,255,0.38)',
                color: '#fff',
                fontSize: 12.5,
                fontWeight: 800,
                whiteSpace: 'nowrap',
              }}
            >
              ✨ AI定制中
            </span>
          )}

          <span
            style={{
              marginLeft: 8,
              padding: '6px 15px',
              borderRadius: 999,
              background: 'rgba(255,255,255,0.18)',
              border:
                '1px solid rgba(255,255,255,0.32)',
              color: '#fff',
              fontSize: 12.5,
              fontWeight: 800,
              whiteSpace: 'nowrap',
            }}
          >
            将融入：第{pageNum}页
          </span>

          <button
            onClick={() => {
              if (!inserting) onClose()
            }}
            title="关闭"
            style={{
              marginLeft: 'auto',
              border: 'none',
              background: 'rgba(255,255,255,0.15)',
              width: 36,
              height: 36,
              borderRadius: 12,
              fontSize: 18,
              cursor: inserting
                ? 'not-allowed'
                : 'pointer',
              color: '#fff',
              flexShrink: 0,
            }}
          >
            ✕
          </button>
        </div>

        <div
          style={{
            flex: 1,
            display: 'flex',
            minHeight: 0,
          }}
        >
          <div
            style={{
              width: 300,
              borderRight: '1px solid ' + C.border,
              display: 'flex',
              flexDirection: 'column',
              flexShrink: 0,
              background:
                'linear-gradient(180deg,#F0FDFA,#F3FAFE)',
            }}
          >
            <div
              style={{
                padding: '14px 14px 12px',
                borderBottom: '1px solid #CCFBF1',
                flexShrink: 0,
              }}
            >
              <div
                style={{
                  fontSize: 13,
                  fontWeight: 850,
                  color: '#115E59',
                }}
              >
                🗺️ 选择地理互动探究
              </div>

              <div
                style={{
                  fontSize: 11.5,
                  color: C.textMuted,
                  marginTop: 6,
                  lineHeight: 1.55,
                }}
              >
                点模板直接使用；点“改编”可让AI在现有结构上生成变种。
              </div>

              <button
                onClick={enterCreateMode}
                style={{
                  width: '100%',
                  marginTop: 10,
                  padding: '9px 0',
                  borderRadius: 12,
                  cursor: 'pointer',
                  border:
                    '1.5px dashed ' +
                    (
                      aiMode === 'create'
                        ? '#0F766E'
                        : '#99F6E4'
                    ),
                  background:
                    aiMode === 'create'
                      ? 'linear-gradient(135deg,#CCFBF1,#EFF6FF)'
                      : '#fff',
                  color: '#0F766E',
                  fontSize: 12.5,
                  fontWeight: 800,
                }}
              >
                ✨ AI新建地理互动
              </button>
            </div>

            <div
              style={{
                flex: 1,
                overflowY: 'auto',
                padding: '12px 12px 16px',
              }}
            >
              {getGeographyLabGroups().map(group => (
                <div
                  key={group.group}
                  style={{ marginBottom: 14 }}
                >
                  <div
                    style={{
                      fontSize: 11.5,
                      fontWeight: 850,
                      color: '#0E7490',
                      padding: '3px 6px 7px',
                      letterSpacing: 0.4,
                    }}
                  >
                    {group.group}
                  </div>

                  {group.items.map(template => {
                    const active =
                      template.id === activeTplId &&
                      !isAI

                    return (
                      <div
                        key={template.id}
                        className="geolab-card"
                        onClick={() =>
                          handlePickTemplate(template)
                        }
                        style={{
                          display: 'flex',
                          gap: 11,
                          alignItems: 'flex-start',
                          padding: '11px 12px 30px',
                          borderRadius: 15,
                          cursor: 'pointer',
                          marginBottom: 8,
                          background: active
                            ? 'linear-gradient(135deg,#CCFBF1,#EFF6FF)'
                            : 'rgba(255,255,255,0.94)',
                          border:
                            '1.5px solid ' +
                            (
                              active
                                ? '#0F766E'
                                : '#D5EEF2'
                            ),
                          boxShadow: active
                            ? '0 8px 22px rgba(15,118,110,0.18)'
                            : '0 2px 8px rgba(14,54,78,0.04)',
                          position: 'relative',
                        }}
                      >
                        <span
                          style={{
                            width: 36,
                            height: 36,
                            borderRadius: 12,
                            background: active
                              ? '#fff'
                              : '#ECFEFF',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontSize: 18,
                            flexShrink: 0,
                          }}
                        >
                          {template.emoji}
                        </span>

                        <div style={{ minWidth: 0 }}>
                          <div
                            style={{
                              fontSize: 13.5,
                              fontWeight: active
                                ? 850
                                : 650,
                              color: active
                                ? '#115E59'
                                : C.textPrimary,
                              lineHeight: 1.35,
                            }}
                          >
                            {template.name}
                          </div>

                          <div
                            style={{
                              fontSize: 11.2,
                              color: C.textMuted,
                              marginTop: 4,
                              lineHeight: 1.5,
                            }}
                          >
                            {template.desc}
                          </div>
                        </div>

                        <button
                          className="geolab-ai-btn"
                          onClick={event => {
                            event.stopPropagation()
                            enterAdaptMode(template)
                          }}
                          title="以此模板为底稿，用自然语言改编"
                          style={{
                            position: 'absolute',
                            right: 9,
                            bottom: 7,
                            padding: '4px 10px',
                            borderRadius: 999,
                            fontSize: 10.5,
                            fontWeight: 800,
                            cursor: 'pointer',
                            border: '1px solid #99F6E4',
                            background: '#fff',
                            color: '#0F766E',
                          }}
                        >
                          🔧 改编
                        </button>
                      </div>
                    )
                  })}
                </div>
              ))}
            </div>
          </div>

          <div
            style={{
              flex: 1,
              display: 'flex',
              minWidth: 0,
            }}
          >
            <div
              style={{
                flex: 1,
                overflow: 'auto',
                padding: '20px 24px',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                minWidth: 0,
                background:
                  'radial-gradient(circle at 50% 0%,#FFFFFF 0%,#F7FBFF 58%,#ECFEFF 100%)',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 9,
                  marginBottom: 14,
                  flexWrap: 'wrap',
                  justifyContent: 'center',
                }}
              >
                <span style={{ fontSize: 20 }}>
                  {isAI ? '✨' : activeTpl.emoji}
                </span>

                <span
                  style={{
                    fontSize: 16,
                    fontWeight: 850,
                    color: C.textPrimary,
                  }}
                >
                  {isAI
                    ? (
                      aiMode === 'adapt'
                        ? activeTpl.name + ' · AI改编'
                        : 'AI新地理互动探究'
                    )
                    : activeTpl.name}
                </span>

                <span
                  style={{
                    fontSize: 12.5,
                    color: C.textMuted,
                  }}
                >
                  {isAI
                    ? (
                      aiCode
                        ? '当前显示AI生成结果'
                        : '右侧描述后生成'
                    )
                    : activeTpl.desc}
                </span>
              </div>

              <div
                style={{
                  padding: 14,
                  background: '#fff',
                  borderRadius: 22,
                  border: '1px solid #CCFBF1',
                  boxShadow:
                    '0 18px 44px rgba(15,118,110,0.12)',
                  flexShrink: 0,
                }}
              >
                <iframe
                  title="地理互动组件预览"
                  sandbox="allow-scripts"
                  srcDoc={previewDoc}
                  style={{
                    width: size.width,
                    height:
                      size.height +
                      (caption.trim() ? 34 : 0),
                    border: 'none',
                    display: 'block',
                    borderRadius: 14,
                    background: 'transparent',
                  }}
                />
              </div>

              <div
                style={{
                  marginTop: 12,
                  fontSize: 12.5,
                  color: '#5B8E89',
                }}
              >
                💡 请先在这里测试滑杆、按钮、图层和自动演示，再融入课件。
              </div>
            </div>

            <div
              style={{
                width: 340,
                borderLeft: '1px solid ' + C.border,
                overflowY: 'auto',
                padding: '18px 18px 22px',
                flexShrink: 0,
                background: '#fff',
              }}
            >
              <div
                style={{
                  padding: '13px 14px',
                  borderRadius: 15,
                  background:
                    'linear-gradient(135deg,#CCFBF1,#EFF6FF)',
                  border: '1px solid #99F6E4',
                  marginBottom: 16,
                }}
              >
                <div
                  style={{
                    fontSize: 13.5,
                    fontWeight: 850,
                    color: '#115E59',
                  }}
                >
                  {isAI
                    ? '✨ AI定制'
                    : '⚙️ 初始参数 / 融入设置'}
                </div>

                <div
                  style={{
                    fontSize: 11.8,
                    color: '#4F7776',
                    lineHeight: 1.65,
                    marginTop: 5,
                  }}
                >
                  {isAI
                    ? 'AI生成的是完整地理互动组件，仍会保留底部课堂控制条和离线运行能力。'
                    : '这里只设置课件中的初始状态；融入后仍可使用组件底部控制条。'}
                </div>
              </div>

              {isAI ? (
                <ExperimentAIPanel
                  target="geography_lab"
                  mode={
                    aiMode as 'adapt' | 'create'
                  }
                  templateName={
                    aiMode === 'adapt'
                      ? activeTpl.name
                      : undefined
                  }
                  baseCode={aiBaseCode}
                  code={aiCode}
                  onCode={setAiCode}
                  onExit={exitAIMode}
                  busyExternal={inserting}
                />
              ) : (
                <>
                  {activeTpl.params.map(param => (
                    <div
                      key={param.key}
                      style={{ marginBottom: 16 }}
                    >
                      {param.type === 'number' && (
                        <>
                          <div
                            style={{
                              display: 'flex',
                              justifyContent:
                                'space-between',
                              alignItems: 'center',
                              marginBottom: 7,
                              gap: 10,
                            }}
                          >
                            <span
                              style={{
                                fontSize: 13,
                                fontWeight: 650,
                                color: C.textPrimary,
                              }}
                            >
                              {param.label}
                            </span>

                            <input
                              type="number"
                              value={Number(params[param.key])}
                              min={param.min}
                              max={param.max}
                              step={param.step}
                              onChange={event => {
                                const raw = parseFloat(
                                  event.target.value,
                                )

                                if (Number.isNaN(raw)) {
                                  return
                                }

                                const min =
                                  param.min ?? raw
                                const max =
                                  param.max ?? raw

                                setParam(
                                  param.key,
                                  Math.min(
                                    max,
                                    Math.max(min, raw),
                                  ),
                                )
                              }}
                              style={{
                                width: 78,
                                padding: '4px 9px',
                                borderRadius: 999,
                                border:
                                  '1.5px solid #99F6E4',
                                background: '#ECFEFF',
                                color: '#115E59',
                                fontWeight: 800,
                                fontSize: 12.5,
                                textAlign: 'center',
                                outline: 'none',
                              }}
                            />
                          </div>

                          <input
                            type="range"
                            className="geolab-range"
                            value={Number(params[param.key])}
                            min={param.min}
                            max={param.max}
                            step={param.step}
                            onChange={event =>
                              setParam(
                                param.key,
                                parseFloat(
                                  event.target.value,
                                ),
                              )}
                          />
                        </>
                      )}

                      {param.type === 'boolean' && (
                        <div
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent:
                              'space-between',
                            gap: 10,
                          }}
                        >
                          <span
                            style={{
                              fontSize: 13,
                              fontWeight: 650,
                              color: C.textPrimary,
                              lineHeight: 1.4,
                            }}
                          >
                            {param.label}
                          </span>

                          <Toggle
                            checked={Boolean(
                              params[param.key],
                            )}
                            onChange={value =>
                              setParam(
                                param.key,
                                value,
                              )}
                          />
                        </div>
                      )}

                      {param.type === 'select' && (
                        <>
                          <div
                            style={{
                              fontSize: 13,
                              fontWeight: 650,
                              color: C.textPrimary,
                              marginBottom: 7,
                            }}
                          >
                            {param.label}
                          </div>

                          <select
                            value={String(
                              params[param.key],
                            )}
                            onChange={event =>
                              setParam(
                                param.key,
                                event.target.value,
                              )}
                            style={{
                              width: '100%',
                              boxSizing: 'border-box',
                              padding: '8px 10px',
                              borderRadius: 11,
                              border:
                                '1.5px solid #99F6E4',
                              background: '#fff',
                              color: '#115E59',
                              fontSize: 12.5,
                              fontWeight: 700,
                              outline: 'none',
                            }}
                          >
                            {(param.options || []).map(
                              option => (
                                <option
                                  key={option.value}
                                  value={option.value}
                                >
                                  {option.label}
                                </option>
                              ),
                            )}
                          </select>
                        </>
                      )}

                      {param.hint && (
                        <div
                          style={{
                            fontSize: 11,
                            color: C.textMuted,
                            marginTop: 5,
                            lineHeight: 1.5,
                          }}
                        >
                          {param.hint}
                        </div>
                      )}
                    </div>
                  ))}
                </>
              )}

              <div
                style={{
                  borderTop:
                    '1px dashed #99F6E4',
                  margin: '18px 0',
                }}
              />

              <div style={{ marginBottom: 16 }}>
                <div
                  style={{
                    fontSize: 13,
                    fontWeight: 650,
                    color: C.textPrimary,
                    marginBottom: 8,
                  }}
                >
                  互动尺寸
                </div>

                <div
                  style={{
                    display: 'flex',
                    borderRadius: 12,
                    overflow: 'hidden',
                    border: '1.5px solid #99F6E4',
                  }}
                >
                  {GEOGRAPHY_LAB_SIZE_PRESETS.map(
                    (preset, index) => {
                      const selected =
                        index === sizeIdx

                      return (
                        <button
                          key={preset.label}
                          onClick={() =>
                            setSizeIdx(index)
                          }
                          style={{
                            flex: 1,
                            padding: '8px 0',
                            border: 'none',
                            cursor: 'pointer',
                            borderRight:
                              index <
                              GEOGRAPHY_LAB_SIZE_PRESETS.length - 1
                                ? '1px solid #99F6E4'
                                : 'none',
                            background: selected
                              ? 'linear-gradient(135deg,#2DD4BF,#0F766E)'
                              : '#fff',
                            color: selected
                              ? '#fff'
                              : C.textSecondary,
                          }}
                        >
                          <div
                            style={{
                              fontSize: 12.5,
                              fontWeight: selected
                                ? 850
                                : 650,
                            }}
                          >
                            {['小', '标准', '大'][index]}
                          </div>

                          <div
                            style={{
                              fontSize: 10,
                              opacity: 0.86,
                              marginTop: 1,
                            }}
                          >
                            {preset.width}×{preset.height}
                          </div>
                        </button>
                      )
                    },
                  )}
                </div>
              </div>

              <div style={{ marginBottom: 16 }}>
                <div
                  style={{
                    fontSize: 13,
                    fontWeight: 650,
                    color: C.textPrimary,
                    marginBottom: 8,
                  }}
                >
                  标注文字（可选）
                </div>

                <input
                  type="text"
                  value={caption}
                  onChange={event =>
                    setCaption(event.target.value)
                  }
                  placeholder="如：拖动经纬度观察半球与纬度带"
                  maxLength={60}
                  style={{
                    width: '100%',
                    boxSizing: 'border-box',
                    padding: '9px 12px',
                    borderRadius: 12,
                    border:
                      '1.5px solid #CCFBF1',
                    fontSize: 13,
                    outline: 'none',
                  }}
                />
              </div>

              <div style={{ marginBottom: 8 }}>
                <div
                  style={{
                    fontSize: 13,
                    fontWeight: 650,
                    color: C.textPrimary,
                    marginBottom: 8,
                  }}
                >
                  融入位置偏好（可选）
                </div>

                <input
                  type="text"
                  value={positionHint}
                  onChange={event =>
                    setPositionHint(
                      event.target.value,
                    )
                  }
                  placeholder="如：放在右侧、替换原来的地图或图表"
                  maxLength={80}
                  style={{
                    width: '100%',
                    boxSizing: 'border-box',
                    padding: '9px 12px',
                    borderRadius: 12,
                    border:
                      '1.5px solid #CCFBF1',
                    fontSize: 13,
                    outline: 'none',
                  }}
                />
              </div>
            </div>
          </div>
        </div>

        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            padding: '14px 24px',
            borderTop: '1px solid ' + C.border,
            flexShrink: 0,
            background: '#F0FDFA',
          }}
        >
          <span
            style={{
              fontSize: 12.5,
              color: C.textMuted,
            }}
          >
            {isAI
              ? 'AI组件仍是纯HTML、SVG、Canvas和原生JavaScript，可离线运行。'
              : '教学简化模型，无在线地图和外部依赖；融入后保留底部课堂控制条。'}
          </span>

          <div
            style={{
              marginLeft: 'auto',
              display: 'flex',
              gap: 10,
            }}
          >
            <button
              onClick={() => {
                if (!inserting) onClose()
              }}
              disabled={inserting}
              style={{
                padding: '10px 24px',
                borderRadius: 13,
                border:
                  '1.5px solid #CCFBF1',
                background: '#fff',
                color: C.textSecondary,
                fontSize: 14,
                fontWeight: 650,
                cursor: inserting
                  ? 'not-allowed'
                  : 'pointer',
              }}
            >
              取消
            </button>

            <button
              onClick={handleInsert}
              disabled={insertDisabled}
              style={{
                padding: '10px 28px',
                borderRadius: 13,
                border: 'none',
                fontSize: 14,
                fontWeight: 850,
                background: insertDisabled
                  ? '#67E8F9'
                  : 'linear-gradient(135deg,#2DD4BF,#0F766E)',
                boxShadow: insertDisabled
                  ? 'none'
                  : '0 8px 22px rgba(15,118,110,0.35)',
                color: '#fff',
                cursor: insertDisabled
                  ? 'not-allowed'
                  : 'pointer',
              }}
            >
              {inserting
                ? '⏳ AI融入中……'
                : '🌍 融入第' + pageNum + '页'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
