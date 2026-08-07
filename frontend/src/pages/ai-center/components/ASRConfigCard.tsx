/**
 * ASRConfigCard — 流式语音识别独立配置卡片
 *
 * 挂载位置：AI管理中心 → 连接配置。
 *
 * 配置目标：
 *   - ASR APP ID：流式语音识别应用；
 *   - ASR Access Token：AES加密存储；
 *   - WebSocket地址：双向流式优化版；
 *   - Resource ID：ASR 2.0小时版；
 *   - 单次录音上限。
 *
 * 安全与业务边界：
 *   1. 不覆盖TTS APP ID和TTS Access Token；
 *   2. Token留空表示保留当前值；
 *   3. 测试连接只完成握手和首包确认，不采集麦克风；
 *   4. 保存成功后后端自动切换为独立ASR凭据。
 */

import {
  useCallback,
  useEffect,
  useState,
} from 'react'
import type {
  CSSProperties,
} from 'react'
import {
  getASRConfig,
  testASRConnection,
  updateASRConfig,
} from '@/api/ai-config'
import type {
  ASRConfigView,
  TestASRResult,
} from '@/api/ai-config'
import { C } from './AICenterConstants'

interface ASRConfigCardProps {
  showToast: (
    message: string,
    type: 'success' | 'error',
  ) => void
}

interface ASRConfigForm {
  app_id: string
  access_token: string
  ws_url: string
  resource_id: string
  max_duration_seconds: string
}

const DEFAULT_WS_URL =
  'wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async'

const DEFAULT_RESOURCE_ID =
  'volc.seedasr.sauc.duration'

const DEFAULT_MAX_DURATION =
  '120'

function buildForm(
  view?: ASRConfigView | null,
): ASRConfigForm {
  return {
    app_id:
      view?.app_id || '',
    access_token: '',
    ws_url:
      view?.ws_url ||
      DEFAULT_WS_URL,
    resource_id:
      view?.resource_id ||
      DEFAULT_RESOURCE_ID,
    max_duration_seconds:
      String(
        view?.max_duration_seconds ||
        Number(DEFAULT_MAX_DURATION),
      ),
  }
}

function validateForm(
  form: ASRConfigForm,
  tokenAlreadySet: boolean,
): string {
  const appID =
    form.app_id.trim()

  if (!appID) {
    return '请填写ASR APP ID'
  }

  if (!/^\d+$/.test(appID)) {
    return 'ASR APP ID只能包含数字'
  }

  if (
    !tokenAlreadySet &&
    !form.access_token.trim()
  ) {
    return '首次保存必须填写ASR Access Token'
  }

  const wsURL =
    form.ws_url.trim()

  if (
    !wsURL.startsWith('wss://')
  ) {
    return 'WebSocket地址必须使用wss://'
  }

  if (!form.resource_id.trim()) {
    return 'Resource ID不能为空'
  }

  const maxDuration =
    Number(
      form.max_duration_seconds,
    )

  if (
    !Number.isInteger(maxDuration) ||
    maxDuration < 5 ||
    maxDuration > 300
  ) {
    return '单次录音上限必须是5至300秒的整数'
  }

  return ''
}

export default function ASRConfigCard({
  showToast,
}: ASRConfigCardProps) {
  const [
    view,
    setView,
  ] =
    useState<ASRConfigView | null>(
      null,
    )

  const [
    form,
    setForm,
  ] =
    useState<ASRConfigForm>(
      buildForm(),
    )

  const [
    loadError,
    setLoadError,
  ] =
    useState<string | null>(
      null,
    )

  const [
    showToken,
    setShowToken,
  ] =
    useState(false)

  const [
    saving,
    setSaving,
  ] =
    useState(false)

  const [
    testing,
    setTesting,
  ] =
    useState(false)

  const [
    testResult,
    setTestResult,
  ] =
    useState<TestASRResult | null>(
      null,
    )

  const loadConfig =
    useCallback(async () => {
      try {
        setLoadError(null)

        const current =
          await getASRConfig()

        setView(current)
        setForm(
          buildForm(current),
        )
      } catch (error: unknown) {
        setLoadError(
          error instanceof Error
            ? error.message
            : 'ASR配置加载失败',
        )
      }
    }, [])

  useEffect(() => {
    void loadConfig()
  }, [loadConfig])

  const handleSave =
    async () => {
      const validationMessage =
        validateForm(
          form,
          Boolean(
            view?.access_token_set,
          ),
        )

      if (validationMessage) {
        showToast(
          validationMessage,
          'error',
        )
        return
      }

      try {
        setSaving(true)
        setTestResult(null)

        const saved =
          await updateASRConfig({
            app_id:
              form.app_id.trim(),
            access_token:
              form.access_token.trim(),
            ws_url:
              form.ws_url.trim(),
            resource_id:
              form.resource_id.trim(),
            max_duration_seconds:
              Number(
                form.max_duration_seconds,
              ),
          })

        setView(saved)
        setForm(
          buildForm(saved),
        )

        showToast(
          'ASR独立配置保存成功',
          'success',
        )
      } catch (error: unknown) {
        showToast(
          error instanceof Error
            ? error.message
            : 'ASR配置保存失败',
          'error',
        )
      } finally {
        setSaving(false)
      }
    }

  const handleTest =
    async () => {
      try {
        setTesting(true)
        setTestResult(null)

        const result =
          await testASRConnection()

        setTestResult(result)

        showToast(
          result.success
            ? 'ASR链路测试成功'
            : 'ASR链路测试失败',
          result.success
            ? 'success'
            : 'error',
        )
      } catch (error: unknown) {
        const message =
          error instanceof Error
            ? error.message
            : '请求失败'

        setTestResult({
          success: false,
          latency_ms: 0,
          resource_id:
            form.resource_id.trim(),
          ws_url:
            form.ws_url.trim(),
          message,
        })

        showToast(
          'ASR链路测试失败',
          'error',
        )
      } finally {
        setTesting(false)
      }
    }

  const inputStyle:
    CSSProperties = {
      width: '100%',
      padding: '10px 14px',
      borderRadius: '10px',
      border:
        `1px solid ${C.border}`,
      fontSize: '14px',
      outline: 'none',
      boxSizing: 'border-box',
      background: C.white,
      fontFamily: 'monospace',
    }

  const configured =
    Boolean(view?.configured)

  return (
    <div
      style={{
        background: C.card,
        borderRadius: '16px',
        border:
          `1px solid ${C.border}`,
        boxShadow:
          '0 2px 8px rgba(0,0,0,0.04)',
        marginTop: '20px',
      }}
    >
      <div
        style={{
          padding: '18px 24px',
          borderBottom:
            `1px solid ${C.border}`,
          display: 'flex',
          alignItems: 'center',
          justifyContent:
            'space-between',
          gap: '16px',
        }}
      >
        <div>
          <div
            style={{
              fontSize: '15px',
              fontWeight: 600,
              color: C.text,
            }}
          >
            🎤 ASR 流式语音识别配置
          </div>

          <div
            style={{
              fontSize: '13px',
              color: C.textSec,
              marginTop: '3px',
            }}
          >
            AI对话输入框的语音转文字通道，与TTS配音凭据完全隔离
          </div>
        </div>

        {view && (
          <div
            style={{
              padding: '5px 12px',
              borderRadius: '16px',
              fontSize: '12px',
              fontWeight: 600,
              flexShrink: 0,
              background:
                configured
                  ? C.successLight
                  : C.dangerLight,
              color:
                configured
                  ? C.success
                  : C.danger,
            }}
          >
            {configured
              ? '✓ 独立ASR凭据已配置'
              : '⚠ 等待独立ASR配置'}
          </div>
        )}
      </div>

      <div
        style={{
          padding: '24px',
        }}
      >
        {loadError && (
          <div
            style={{
              padding: '12px 16px',
              borderRadius: '10px',
              marginBottom: '16px',
              background:
                C.dangerLight,
              border:
                '1px solid rgba(239,68,68,0.25)',
              fontSize: '13px',
              color: C.danger,
              display: 'flex',
              alignItems: 'center',
              justifyContent:
                'space-between',
              gap: '12px',
            }}
          >
            <span>
              ⚠ {loadError}
            </span>

            <button
              type="button"
              onClick={() =>
                void loadConfig()
              }
              style={{
                padding: '4px 12px',
                borderRadius: '6px',
                border:
                  `1px solid ${C.border}`,
                background: C.white,
                fontSize: '12px',
                color: C.textSec,
                cursor: 'pointer',
              }}
            >
              重试
            </button>
          </div>
        )}

        {!view?.using_separate_credentials && (
          <div
            style={{
              padding: '12px 16px',
              borderRadius: '10px',
              marginBottom: '16px',
              background:
                'rgba(245,158,11,0.08)',
              border:
                '1px solid rgba(245,158,11,0.25)',
              color: '#92400E',
              fontSize: '13px',
              lineHeight: 1.6,
            }}
          >
            当前语音识别仍在兼容复用TTS凭据。保存本卡片后将切换为独立ASR应用，
            不会修改现有课件配音配置。
          </div>
        )}

        <div
          style={{
            display: 'grid',
            gridTemplateColumns:
              '1fr 180px',
            gap: '14px',
            marginBottom: '16px',
          }}
        >
          <div>
            <label
              style={{
                display: 'block',
                fontSize: '13px',
                fontWeight: 600,
                color: C.text,
                marginBottom: '6px',
              }}
            >
              服务
            </label>

            <input
              value={
                view?.service_name ||
                '豆包流式语音识别模型2.0'
              }
              readOnly
              style={{
                ...inputStyle,
                background: C.bg,
                color: C.textSec,
              }}
            />
          </div>

          <div>
            <label
              style={{
                display: 'block',
                fontSize: '13px',
                fontWeight: 600,
                color: C.text,
                marginBottom: '6px',
              }}
            >
              计费类型
            </label>

            <input
              value={
                view?.billing_mode ||
                '小时版'
              }
              readOnly
              style={{
                ...inputStyle,
                background: C.bg,
                color: C.textSec,
              }}
            />
          </div>
        </div>

        <div
          style={{
            marginBottom: '16px',
          }}
        >
          <label
            style={{
              display: 'block',
              fontSize: '13px',
              fontWeight: 600,
              color: C.text,
              marginBottom: '6px',
            }}
          >
            ASR APP ID
          </label>

          <input
            value={form.app_id}
            onChange={(event) =>
              setForm(
                (previous) => ({
                  ...previous,
                  app_id:
                    event.target.value,
                }),
              )
            }
            inputMode="numeric"
            placeholder="例如：1676345172"
            style={inputStyle}
            onFocus={(event) => {
              event.currentTarget
                .style.borderColor =
                C.primary
            }}
            onBlur={(event) => {
              event.currentTarget
                .style.borderColor =
                C.border
            }}
          />
        </div>

        <div
          style={{
            marginBottom: '16px',
          }}
        >
          <label
            style={{
              display: 'block',
              fontSize: '13px',
              fontWeight: 600,
              color: C.text,
              marginBottom: '6px',
            }}
          >
            ASR Access Token

            {view?.access_token_set && (
              <span
                style={{
                  fontWeight: 400,
                  color: C.textMuted,
                  marginLeft: '8px',
                  fontSize: '12px',
                }}
              >
                当前：{view.access_token}
                （留空不修改）
              </span>
            )}
          </label>

          <div
            style={{
              position: 'relative',
            }}
          >
            <input
              type={
                showToken
                  ? 'text'
                  : 'password'
              }
              value={
                form.access_token
              }
              onChange={(event) =>
                setForm(
                  (previous) => ({
                    ...previous,
                    access_token:
                      event.target.value,
                  }),
                )
              }
              placeholder={
                view?.access_token_set
                  ? '留空表示不修改'
                  : '请输入default应用的Access Token'
              }
              style={{
                ...inputStyle,
                padding:
                  '10px 44px 10px 14px',
              }}
            />

            <button
              type="button"
              onClick={() =>
                setShowToken(
                  (previous) =>
                    !previous,
                )
              }
              style={{
                position: 'absolute',
                right: '12px',
                top: '50%',
                transform:
                  'translateY(-50%)',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                color: C.textMuted,
                fontSize: '16px',
              }}
            >
              {showToken
                ? '🙈'
                : '👁'}
            </button>
          </div>

          <div
            style={{
              fontSize: '12px',
              color: C.textMuted,
              marginTop: '3px',
            }}
          >
            Token由后端AES加密存储；Secret Key不需要填写
          </div>
        </div>

        <div
          style={{
            marginBottom: '16px',
          }}
        >
          <label
            style={{
              display: 'block',
              fontSize: '13px',
              fontWeight: 600,
              color: C.text,
              marginBottom: '6px',
            }}
          >
            WebSocket 地址
          </label>

          <input
            value={form.ws_url}
            onChange={(event) =>
              setForm(
                (previous) => ({
                  ...previous,
                  ws_url:
                    event.target.value,
                }),
              )
            }
            style={inputStyle}
          />
        </div>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns:
              '1fr 180px',
            gap: '14px',
            marginBottom: '20px',
          }}
        >
          <div>
            <label
              style={{
                display: 'block',
                fontSize: '13px',
                fontWeight: 600,
                color: C.text,
                marginBottom: '6px',
              }}
            >
              Resource ID
            </label>

            <input
              value={
                form.resource_id
              }
              onChange={(event) =>
                setForm(
                  (previous) => ({
                    ...previous,
                    resource_id:
                      event.target.value,
                  }),
                )
              }
              style={inputStyle}
            />
          </div>

          <div>
            <label
              style={{
                display: 'block',
                fontSize: '13px',
                fontWeight: 600,
                color: C.text,
                marginBottom: '6px',
              }}
            >
              单次录音上限
            </label>

            <div
              style={{
                position: 'relative',
              }}
            >
              <input
                type="number"
                min="5"
                max="300"
                step="1"
                value={
                  form.max_duration_seconds
                }
                onChange={(event) =>
                  setForm(
                    (previous) => ({
                      ...previous,
                      max_duration_seconds:
                        event.target.value,
                    }),
                  )
                }
                style={{
                  ...inputStyle,
                  paddingRight: '42px',
                }}
              />

              <span
                style={{
                  position: 'absolute',
                  right: '13px',
                  top: '50%',
                  transform:
                    'translateY(-50%)',
                  color: C.textMuted,
                  fontSize: '12px',
                }}
              >
                秒
              </span>
            </div>
          </div>
        </div>

        {testResult && (
          <div
            style={{
              padding: '16px',
              borderRadius: '12px',
              marginBottom: '20px',
              background:
                testResult.success
                  ? C.successLight
                  : C.dangerLight,
              border:
                `1px solid ${
                  testResult.success
                    ? 'rgba(16,185,129,0.3)'
                    : 'rgba(239,68,68,0.3)'
                }`,
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '10px',
                marginBottom: '8px',
                flexWrap: 'wrap',
              }}
            >
              <span>
                {testResult.success
                  ? '✅'
                  : '❌'}
              </span>

              <span
                style={{
                  fontWeight: 600,
                  color:
                    testResult.success
                      ? C.success
                      : C.danger,
                }}
              >
                {testResult.success
                  ? 'ASR链路畅通'
                  : 'ASR链路失败'}
              </span>

              {testResult.latency_ms > 0 && (
                <span
                  style={{
                    padding: '2px 8px',
                    borderRadius: '6px',
                    fontSize: '12px',
                    background:
                      'rgba(0,0,0,0.06)',
                    color: C.textSec,
                  }}
                >
                  {testResult.latency_ms}ms
                </span>
              )}
            </div>

            <div
              style={{
                fontSize: '13px',
                color: C.text,
              }}
            >
              {testResult.message}
            </div>

            <div
              style={{
                fontSize: '12px',
                color: C.textMuted,
                marginTop: '5px',
                overflowWrap:
                  'anywhere',
              }}
            >
              resource：
              {testResult.resource_id}
            </div>

            {testResult.log_id && (
              <div
                style={{
                  fontSize: '12px',
                  color: C.textMuted,
                  marginTop: '3px',
                  overflowWrap:
                    'anywhere',
                }}
              >
                火山 Log ID：
                {testResult.log_id}
              </div>
            )}

            <button
              type="button"
              onClick={() =>
                setTestResult(null)
              }
              style={{
                marginTop: '10px',
                padding: '4px 12px',
                borderRadius: '6px',
                border:
                  `1px solid ${C.border}`,
                background: C.white,
                fontSize: '12px',
                color: C.textSec,
                cursor: 'pointer',
              }}
            >
              关闭
            </button>
          </div>
        )}

        <div
          style={{
            display: 'flex',
            gap: '10px',
            justifyContent:
              'flex-end',
          }}
        >
          <button
            type="button"
            onClick={() =>
              void handleTest()
            }
            disabled={
              testing ||
              !configured
            }
            title={
              configured
                ? '测试ASR连接'
                : '请先保存独立ASR配置'
            }
            style={{
              padding: '10px 20px',
              borderRadius: '10px',
              border: 'none',
              background:
                testing ||
                !configured
                  ? C.textMuted
                  : 'linear-gradient(135deg,#F59E0B,#D97706)',
              color: '#fff',
              fontSize: '14px',
              fontWeight: 600,
              cursor:
                testing ||
                !configured
                  ? 'not-allowed'
                  : 'pointer',
            }}
          >
            {testing
              ? '测试中...'
              : '🔌 测试ASR连接'}
          </button>

          <button
            type="button"
            onClick={() =>
              void handleSave()
            }
            disabled={saving}
            style={{
              padding: '10px 24px',
              borderRadius: '10px',
              border: 'none',
              background:
                saving
                  ? C.textMuted
                  : `linear-gradient(135deg,${C.primary},#7C3AED)`,
              color: '#fff',
              fontSize: '14px',
              fontWeight: 600,
              cursor:
                saving
                  ? 'not-allowed'
                  : 'pointer',
            }}
          >
            {saving
              ? '保存中...'
              : '💾 保存ASR配置'}
          </button>
        </div>
      </div>
    </div>
  )
}
