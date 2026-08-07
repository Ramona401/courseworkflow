/**
 * AI美术风格工作室动态加载外壳。
 *
 * 工作室包含多轮对话、上传、IAOCI展示和三类预览，
 * 体积明显大于普通风格模板选择器。
 *
 * 使用React.lazy后：
 *   - 老师未打开工作室时不下载重型内容；
 *   - 课件工坊主页面首包不再直接包含整个工作室；
 *   - 选风格步骤和全自动装配入口共用同一个按需分块。
 */

import {
  lazy,
  Suspense,
} from 'react'
import type {
  StyleStudioModalContentProps,
} from './StyleStudioModalContent'

const LazyStyleStudioModalContent =
  lazy(
    () =>
      import(
        './StyleStudioModalContent'
      ),
  )

export type StyleStudioModalProps =
  StyleStudioModalContentProps

export default function StyleStudioModal(
  props: StyleStudioModalProps,
) {
  if (!props.open) {
    return null
  }

  return (
    <Suspense
      fallback={
        <div
          style={{
            position: 'fixed',
            inset: 0,
            zIndex: 100500,
            padding: 20,
            background:
              'rgba(15,23,42,0.62)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <div
            style={{
              width: 320,
              padding: '34px 24px',
              borderRadius: 16,
              background: '#fff',
              boxShadow:
                '0 24px 70px rgba(15,23,42,0.32)',
              textAlign: 'center',
              color: '#6B7280',
              fontSize: 13,
            }}
          >
            <div
              style={{
                fontSize: 38,
                marginBottom: 10,
              }}
            >
              ✨
            </div>
            正在加载AI美术风格工作室...
          </div>
        </div>
      }
    >
      <LazyStyleStudioModalContent
        {...props}
      />
    </Suspense>
  )
}
