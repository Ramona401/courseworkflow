/**
 * IdentityAccountLinkCard — TE-DNA Identity Phase 1账号关联入口。
 *
 * 边界：
 * - 仅供已经登录TE-DNA的当前账号主动Link/Unlink；
 * - 浏览器不提交users.id、global_person_id或其它本地身份字段；
 * - 只从后端取得经过校验的Identity授权URL后进行顶层导航；
 * - 不实现Identity登录，不读写TE-DNA JWT，不创建Session。
 */

import { useState } from 'react'

import {
  getIdentityLinkAuthorizationURL,
  getIdentityUnlinkAuthorizationURL,
} from '@/api/identity-account-link'

type IdentityAction =
  | 'link'
  | 'unlink'

interface IdentityAccountLinkCardProps {
  onError: (message: string) => void
}

const COLORS = {
  primary: '#4F7BE8',
  danger: '#EF4444',
  text: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  background: '#F9FAFB',
  white: '#FFFFFF',
}

export default function IdentityAccountLinkCard({
  onError,
}: IdentityAccountLinkCardProps) {
  const [
    activeAction,
    setActiveAction,
  ] = useState<IdentityAction | null>(
    null,
  )

  const busy =
    activeAction !== null

  const startAuthorization =
    async (
      action: IdentityAction,
    ) => {
      if (busy) {
        return
      }

      setActiveAction(action)

      try {
        const authorizationURL =
          action === 'link'
            ? await getIdentityLinkAuthorizationURL()
            : await getIdentityUnlinkAuthorizationURL()

        // API层已将目标Origin严格限制为生产Identity Center。
        window.location.assign(
          authorizationURL,
        )
      } catch (error: unknown) {
        setActiveAction(null)

        onError(
          error instanceof Error
            ? error.message
            : 'Identity账号关联服务暂时不可用',
        )
      }
    }

  return (
    <section
      style={{
        background: COLORS.white,
        borderRadius: '16px',
        border:
          `1px solid ${COLORS.border}`,
        padding: '24px 28px',
        marginBottom: '20px',
        boxShadow:
          '0 1px 4px rgba(0,0,0,0.04)',
      }}
    >
      <h2
        style={{
          margin: '0 0 8px',
          fontSize: '16px',
          fontWeight: 600,
          color: COLORS.text,
        }}
      >
        Identity Center账号关联
      </h2>

      <p
        style={{
          margin: '0 0 18px',
          fontSize: '13px',
          lineHeight: 1.7,
          color: COLORS.textSecondary,
        }}
      >
        将当前已登录的TE-DNA账号与Identity Center人员身份建立或解除映射。
        此操作不会改变TE-DNA密码、登录JWT、角色或本地权限。
      </p>

      <div
        style={{
          padding: '16px',
          borderRadius: '12px',
          border:
            `1px solid ${COLORS.border}`,
          background:
            COLORS.background,
        }}
      >
        <div
          style={{
            marginBottom: '14px',
            fontSize: '13px',
            lineHeight: 1.7,
            color: COLORS.textMuted,
          }}
        >
          TE-DNA不会在浏览器中提交或展示global_person_id和本地users.id。
          授权结束后只显示非敏感的关联结果。
        </div>

        <div
          style={{
            display: 'flex',
            gap: '10px',
            flexWrap: 'wrap',
          }}
        >
          <button
            type="button"
            disabled={busy}
            onClick={() =>
              void startAuthorization(
                'link',
              )
            }
            style={{
              flex: '1 1 220px',
              padding: '10px 18px',
              border: 'none',
              borderRadius: '9px',
              background:
                busy
                  ? COLORS.textMuted
                  : 'linear-gradient(135deg,#4F7BE8,#6366F1)',
              color: COLORS.white,
              fontSize: '14px',
              fontWeight: 600,
              cursor:
                busy
                  ? 'not-allowed'
                  : 'pointer',
            }}
          >
            {
              activeAction === 'link'
                ? '正在前往Identity Center...'
                : '关联Identity Center'
            }
          </button>

          <button
            type="button"
            disabled={busy}
            onClick={() =>
              void startAuthorization(
                'unlink',
              )
            }
            style={{
              flex: '1 1 220px',
              padding: '10px 18px',
              border:
                `1px solid ${COLORS.danger}`,
              borderRadius: '9px',
              background: COLORS.white,
              color: COLORS.danger,
              fontSize: '14px',
              fontWeight: 600,
              cursor:
                busy
                  ? 'not-allowed'
                  : 'pointer',
              opacity:
                busy
                  ? 0.55
                  : 1,
            }}
          >
            {
              activeAction === 'unlink'
                ? '正在前往Identity Center...'
                : '解除Identity关联'
            }
          </button>
        </div>
      </div>
    </section>
  )
}
