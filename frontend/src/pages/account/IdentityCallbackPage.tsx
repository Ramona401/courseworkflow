/**
 * IdentityCallbackPage — Identity Phase 1账号关联结果页。
 *
 * 安全边界：
 * - 只消费后端固定redirect中的非敏感结果；
 * - 不接收global_person_id、local_account_id、OIDC code、nonce或Secret；
 * - 不调用Identity登录，不换发TE-DNA JWT，不建立任何Session；
 * - success与error结果严格互斥，未知或混合参数统一按无效结果处理。
 */

import {
  useLocation,
  useNavigate,
} from 'react-router-dom'

type IdentityOperation =
  | 'link'
  | 'unlink'

type IdentityState =
  | 'linked'
  | 'unlinked'

interface IdentityCallbackSuccess {
  kind: 'success'
  operation: IdentityOperation
  state: IdentityState
  changed: boolean
}

interface IdentityCallbackFailure {
  kind: 'error'
  message: string
}

type IdentityCallbackResult =
  | IdentityCallbackSuccess
  | IdentityCallbackFailure

const COLORS = {
  primary: '#4F7BE8',
  success: '#10B981',
  danger: '#EF4444',
  text: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  white: '#FFFFFF',
}

const SAFE_ERROR_MESSAGES:
  Record<string, string> = {
    IDENTITY_CALLBACK_INVALID:
      '授权回调无效，请返回个人中心重新操作。',

    IDENTITY_AUTHORIZATION_DENIED:
      '本次Identity授权已取消。',

    IDENTITY_FLOW_MISSING:
      '授权流程已经失效，请返回个人中心重新操作。',

    IDENTITY_FLOW_INVALID:
      '授权流程无效，请返回个人中心重新操作。',

    IDENTITY_STATE_MISMATCH:
      '授权状态校验失败，请返回个人中心重新操作。',

    IDENTITY_INVALID_OPERATION:
      '本次账号关联操作无效。',

    IDENTITY_UNAVAILABLE:
      'Identity账号关联服务暂时不可用，请稍后重试。',

    IDENTITY_NOT_INITIALIZED:
      'Identity账号关联服务暂未完成配置。',

    IDENTITY_CONTEXT_INVALID:
      '本次账号关联上下文无效，请重新操作。',

    IDENTITY_UPSTREAM_ERROR:
      'Identity服务暂时异常，请稍后重试。',

    IDENTITY_PROTOCOL_ERROR:
      'Identity协议校验失败，请重新操作。',

    IDENTITY_PERSON_UNAVAILABLE:
      '当前Identity账号暂不可用于账号关联。',

    IDENTITY_LINK_CONFLICT:
      '当前Identity账号或TE-DNA账号已存在其它关联。',

    IDENTITY_LINK_STATE_CHANGED:
      '账号关联状态已经变化，请刷新后重新操作。',

    IDENTITY_OPERATION_FAILED:
      '账号关联操作未完成，请稍后重试。',

    ACCOUNT_UNAVAILABLE:
      '当前TE-DNA账号不可用于账号关联。',

    ACCOUNT_DISABLED:
      '当前TE-DNA账号已被禁用。',

    UNAUTHORIZED:
      '当前TE-DNA登录状态无效，请重新登录。',

    DATABASE_ERROR:
      'TE-DNA账号状态暂时无法确认，请稍后重试。',
  }

function hasOnlyKeys(
  params: URLSearchParams,
  allowed: readonly string[],
): boolean {
  const allowedSet =
    new Set(allowed)

  for (const key of params.keys()) {
    if (!allowedSet.has(key)) {
      return false
    }
  }

  for (const key of allowed) {
    if (params.getAll(key).length !== 1) {
      return false
    }
  }

  return true
}

function parseIdentityCallbackResult(
  search: string,
): IdentityCallbackResult {
  const params =
    new URLSearchParams(search)

  const hasError =
    params.has('error')

  const hasSuccessField =
    params.has('operation') ||
    params.has('state') ||
    params.has('changed')

  if (hasError && hasSuccessField) {
    return {
      kind: 'error',
      message:
        '账号关联结果无效，请返回个人中心重新操作。',
    }
  }

  if (hasError) {
    if (
      !hasOnlyKeys(
        params,
        ['error'],
      )
    ) {
      return {
        kind: 'error',
        message:
          '账号关联结果无效，请返回个人中心重新操作。',
      }
    }

    const errorCode =
      params.get('error') || ''

    return {
      kind: 'error',
      message:
        SAFE_ERROR_MESSAGES[errorCode] ||
        '账号关联操作未完成，请返回个人中心重新操作。',
    }
  }

  if (
    !hasOnlyKeys(
      params,
      [
        'operation',
        'state',
        'changed',
      ],
    )
  ) {
    return {
      kind: 'error',
      message:
        '账号关联结果无效，请返回个人中心重新操作。',
    }
  }

  const operation =
    params.get('operation')

  const state =
    params.get('state')

  const changedRaw =
    params.get('changed')

  const validPair =
    (
      operation === 'link' &&
      state === 'linked'
    ) ||
    (
      operation === 'unlink' &&
      state === 'unlinked'
    )

  if (
    !validPair ||
    (
      changedRaw !== 'true' &&
      changedRaw !== 'false'
    )
  ) {
    return {
      kind: 'error',
      message:
        '账号关联结果无效，请返回个人中心重新操作。',
    }
  }

  return {
    kind: 'success',
    operation:
      operation as IdentityOperation,
    state:
      state as IdentityState,
    changed:
      changedRaw === 'true',
  }
}

export default function IdentityCallbackPage() {
  const navigate =
    useNavigate()

  const location =
    useLocation()

  const result =
    parseIdentityCallbackResult(
      location.search,
    )

  const success =
    result.kind === 'success'

  const title =
    success
      ? result.operation === 'link'
        ? 'Identity账号关联完成'
        : 'Identity账号解除关联完成'
      : 'Identity账号关联未完成'

  const description =
    success
      ? result.changed
        ? result.operation === 'link'
          ? 'Identity Center与当前TE-DNA账号的关联已经建立。'
          : 'Identity Center与当前TE-DNA账号的关联已经解除。'
        : result.operation === 'link'
          ? '当前账号已经处于关联状态，无需重复修改。'
          : '当前账号已经处于未关联状态，无需重复修改。'
      : result.message

  return (
    <div
      style={{
        minHeight: '100vh',
        background:
          'linear-gradient(135deg,#F0F4FF 0%,#FAFBFC 50%,#F0FDF4 100%)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '24px',
        boxSizing: 'border-box',
      }}
    >
      <main
        style={{
          width: '100%',
          maxWidth: '520px',
          background: COLORS.white,
          border:
            `1px solid ${COLORS.border}`,
          borderRadius: '20px',
          padding: '36px 32px',
          boxShadow:
            '0 12px 36px rgba(31,41,55,0.08)',
          textAlign: 'center',
        }}
      >
        <div
          style={{
            width: '64px',
            height: '64px',
            margin: '0 auto 20px',
            borderRadius: '50%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '30px',
            background:
              success
                ? 'rgba(16,185,129,0.10)'
                : 'rgba(239,68,68,0.10)',
            color:
              success
                ? COLORS.success
                : COLORS.danger,
          }}
          aria-hidden="true"
        >
          {success ? '✓' : '!'}
        </div>

        <h1
          style={{
            margin: '0 0 12px',
            fontSize: '22px',
            lineHeight: 1.35,
            color: COLORS.text,
          }}
        >
          {title}
        </h1>

        <p
          style={{
            margin: '0',
            fontSize: '14px',
            lineHeight: 1.8,
            color: COLORS.textSecondary,
          }}
        >
          {description}
        </p>

        {success && (
          <div
            style={{
              marginTop: '20px',
              padding: '14px 16px',
              borderRadius: '12px',
              border:
                `1px solid ${COLORS.border}`,
              background: '#F9FAFB',
              textAlign: 'left',
            }}
          >
            <div
              style={{
                fontSize: '13px',
                lineHeight: 1.7,
                color: COLORS.textMuted,
              }}
            >
              本页面仅展示账号关联结果。
              TE-DNA现有登录状态、角色和本地权限体系不会因此改变。
            </div>
          </div>
        )}

        <div
          style={{
            marginTop: '28px',
            display: 'flex',
            gap: '12px',
          }}
        >
          <button
            type="button"
            onClick={() =>
              navigate(
                '/account',
                {
                  replace: true,
                },
              )
            }
            style={{
              flex: 1,
              padding: '11px 18px',
              border: 'none',
              borderRadius: '10px',
              background:
                'linear-gradient(135deg,#4F7BE8,#6366F1)',
              color: COLORS.white,
              fontSize: '14px',
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            返回个人中心
          </button>

          <button
            type="button"
            onClick={() =>
              navigate(
                '/',
                {
                  replace: true,
                },
              )
            }
            style={{
              flex: 1,
              padding: '11px 18px',
              border:
                `1px solid ${COLORS.border}`,
              borderRadius: '10px',
              background: COLORS.white,
              color: COLORS.textSecondary,
              fontSize: '14px',
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            返回首页
          </button>
        </div>
      </main>
    </div>
  )
}
