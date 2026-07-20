/**
 * ProtectedTextbookImage — 课本图片统一鉴权加载组件
 *
 * 上下文15安全规则：
 *   - 不使用后端返回的任意图片URL直接设置img.src；
 *   - 只根据正式课本ID请求固定鉴权图片端点；
 *   - 请求携带当前JWT，并且禁用HTTP缓存；
 *   - 后端实时复核用户教育域、课本active状态和文件路径；
 *   - 图片只存在于浏览器临时Blob URL；
 *   - 组件卸载、课本ID变化或图片解码失败时立即撤销Blob URL；
 *   - 非K12、伪造ID、归档页面或失效登录态均只显示占位内容。
 */

import {
  useEffect,
  useState,
} from 'react'

interface ProtectedTextbookImageProps {
  textbookId: string
  alt: string
  style?: React.CSSProperties
  className?: string
  title?: string

  /**
   * 图片加载中或不可用时显示的内容。
   *
   * 调用方可以根据卡片、缩略图或大图预览场景
   * 传入不同占位文字或图标。
   */
  fallback?: React.ReactNode
}

export default function ProtectedTextbookImage({
  textbookId,
  alt,
  style,
  className,
  title,
  fallback,
}: ProtectedTextbookImageProps) {
  const [blobURL, setBlobURL] =
    useState('')

  const [failed, setFailed] =
    useState(false)

  useEffect(() => {
    const normalizedID =
      textbookId.trim()

    setBlobURL('')
    setFailed(false)

    if (!normalizedID) {
      setFailed(true)
      return
    }

    const controller =
      new AbortController()

    let disposed = false
    let createdURL = ''

    const loadImage = async () => {
      try {
        const token =
          localStorage.getItem('token')

        const response = await fetch(
          `/api/v1/lesson-plans/textbooks/${
            encodeURIComponent(normalizedID)
          }/image`,
          {
            method: 'GET',
            credentials: 'same-origin',
            cache: 'no-store',
            signal: controller.signal,
            headers: token
              ? {
                  Authorization:
                    `Bearer ${token}`,
                }
              : undefined,
          },
        )

        if (!response.ok) {
          throw new Error(
            `课本图片请求失败：${response.status}`,
          )
        }

        const blob =
          await response.blob()

        if (
          !blob.type.toLowerCase()
            .startsWith('image/')
        ) {
          throw new Error(
            '课本图片响应格式无效',
          )
        }

        createdURL =
          URL.createObjectURL(blob)

        if (disposed) {
          URL.revokeObjectURL(
            createdURL,
          )
          createdURL = ''
          return
        }

        setBlobURL(createdURL)
      } catch (error) {
        if (
          error instanceof DOMException &&
          error.name === 'AbortError'
        ) {
          return
        }

        console.error(
          '加载鉴权课本图片失败:',
          error,
        )

        if (!disposed) {
          setFailed(true)
        }
      }
    }

    void loadImage()

    return () => {
      disposed = true
      controller.abort()

      if (createdURL) {
        URL.revokeObjectURL(
          createdURL,
        )
      }
    }
  }, [textbookId])

  const handleImageError = () => {
    setFailed(true)

    setBlobURL(currentURL => {
      if (currentURL) {
        URL.revokeObjectURL(
          currentURL,
        )
      }

      return ''
    })
  }

  if (!blobURL) {
    return (
      <div
        className={className}
        title={title || alt}
        aria-label={alt}
        aria-busy={!failed}
        style={{
          ...style,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          overflow: 'hidden',
          background: '#F9FAFB',
          color: '#9CA3AF',
          fontSize: '11px',
          textAlign: 'center',
        }}
      >
        {fallback ??
          (failed
            ? '图片不可用'
            : '加载中…')}
      </div>
    )
  }

  return (
    <img
      src={blobURL}
      alt={alt}
      title={title}
      className={className}
      style={style}
      draggable={false}
      onError={handleImageError}
    />
  )
}
