/**
 * identity-account-link.ts — TE-DNA Identity Phase 1账号关联API。
 *
 * 边界：
 * - 只为已经登录TE-DNA的本地账号发起Link/Unlink授权；
 * - 本地账号ID由后端JWT claims决定，前端绝不提交user_id/local_account_id；
 * - 不实现Identity登录，不接收或写入TE-DNA JWT，不建立Session。
 */

import client from './client'
import type { ApiResponse } from './client'

const IDENTITY_ORIGIN =
  'https://id.pkuailab.com'

interface IdentityAuthorizationResponse {
  authorization_url: string
}

function validateAuthorizationURL(
  value: unknown,
): string {
  if (
    typeof value !== 'string' ||
    value.trim() !== value ||
    value === ''
  ) {
    throw new Error(
      'Identity授权地址无效，请稍后重试',
    )
  }

  let parsed: URL

  try {
    parsed = new URL(value)
  } catch {
    throw new Error(
      'Identity授权地址无效，请稍后重试',
    )
  }

  if (
    parsed.origin !== IDENTITY_ORIGIN ||
    parsed.protocol !== 'https:' ||
    parsed.username !== '' ||
    parsed.password !== ''
  ) {
    throw new Error(
      'Identity授权地址校验失败',
    )
  }

  return parsed.toString()
}

async function getAuthorizationURL(
  path:
    | '/auth/identity/link-url'
    | '/auth/identity/unlink-url',
): Promise<string> {
  const response =
    await client.get<
      ApiResponse<IdentityAuthorizationResponse>
    >(path)

  return validateAuthorizationURL(
    response.data.data?.authorization_url,
  )
}

export function getIdentityLinkAuthorizationURL():
  Promise<string> {
  return getAuthorizationURL(
    '/auth/identity/link-url',
  )
}

export function getIdentityUnlinkAuthorizationURL():
  Promise<string> {
  return getAuthorizationURL(
    '/auth/identity/unlink-url',
  )
}
