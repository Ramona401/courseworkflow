/**
 * BaseDataPage — 基础数据管理独立页面
 *
 * 三个基础数据维度：
 *   1. 统一课程定义；
 *   2. K12课程大纲；
 *   3. 组织教育域只读清单。
 *
 * 组织教育域Tab：
 *   - 所有可进入基础数据页的admin只读查看；
 *   - 区域固定为mixed；
 *   - 学校教育域创建后不可修改。
 */

import { useState } from 'react'
import {
  useLocation,
  useNavigate,
} from 'react-router-dom'
import { C } from '@/pages/admin/components/adminConstants'
import {
  SubjectsPanel,
} from '@/pages/admin/components/SubjectsPanel'
import CourseOutlinesPage from '@/pages/lesson-plans/course-outlines/CourseOutlinesPage'
import OrganizationEducationDomainsPanel from './OrganizationEducationDomainsPanel'

type BaseDataTab =
  | 'subjects'
  | 'outlines'
  | 'domains'

const subTabs: {
  key: BaseDataTab
  label: string
}[] = [
  {
    key: 'subjects',
    label: '📚 课程定义',
  },
  {
    key: 'outlines',
    label: '📖 K12课程大纲',
  },
  {
    key: 'domains',
    label: '🏫 组织教育域',
  },
]

export default function BaseDataPage() {
  const navigate = useNavigate()
  const location = useLocation()

  const fromPath =
    (location.state as { from?: string })?.from
    || '/'

  const [sub, setSub] =
    useState<BaseDataTab>('subjects')

  return (
    <div style={{
      minHeight: '100vh',
      background:
        'linear-gradient(135deg,#EEF2FF 0%,#FAFBFC 50%,#F0FDF4 100%)',
    }}>
      <header style={{
        height: '64px',
        position: 'sticky',
        top: 0,
        zIndex: 100,
        background: 'rgba(255,255,255,0.88)',
        backdropFilter: 'blur(20px)',
        borderBottom: `1px solid ${C.border}`,
        display: 'flex',
        alignItems: 'center',
        padding: '0 32px',
        gap: '16px',
      }}>
        <button
          onClick={() => navigate(fromPath)}
          style={{
            padding: '8px 16px',
            borderRadius: '8px',
            border: `1px solid ${C.border}`,
            background: C.white,
            fontSize: '14px',
            color: C.textSec,
            cursor: 'pointer',
          }}
          onMouseEnter={event => {
            event.currentTarget.style.background = C.bg
          }}
          onMouseLeave={event => {
            event.currentTarget.style.background = C.white
          }}
        >
          {'<- 返回'}
        </button>

        <div style={{
          flex: 1,
          textAlign: 'center',
        }}>
          <h1 style={{
            fontSize: '18px',
            fontWeight: 700,
            color: C.text,
            margin: 0,
          }}>
            📚 基础数据管理
          </h1>

          <div style={{
            fontSize: '12px',
            color: C.textMuted,
            marginTop: '2px',
          }}>
            管理课程定义和K12课程大纲，查看组织教育域
          </div>
        </div>

        <div style={{ width: '92px' }} />
      </header>

      <div style={{
        maxWidth: '1400px',
        margin: '0 auto',
        padding: '24px',
      }}>
        <div style={{
          display: 'flex',
          gap: '4px',
          borderBottom: `1px solid ${C.border}`,
          marginBottom: '20px',
        }}>
          {subTabs.map(tab => {
            const active = sub === tab.key

            return (
              <button
                key={tab.key}
                onClick={() => setSub(tab.key)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                  padding: '10px 18px',
                  border: 'none',
                  background: 'transparent',
                  cursor: 'pointer',
                  fontSize: '14px',
                  fontWeight: active ? 600 : 400,
                  color: active
                    ? C.primary
                    : C.textSec,
                  borderBottom: active
                    ? `2px solid ${C.primary}`
                    : '2px solid transparent',
                  marginBottom: '-1px',
                  transition: 'all 160ms ease',
                }}
              >
                {tab.label}
              </button>
            )
          })}
        </div>

        {sub === 'subjects' && (
          <SubjectsPanel />
        )}

        {sub === 'outlines' && (
          <div>
            <div style={{
              fontSize: '16px',
              fontWeight: 700,
              color: C.text,
            }}>
              📖 K12课程大纲
            </div>

            <div style={{
              fontSize: '12px',
              color: C.textMuted,
              marginTop: '4px',
              marginBottom: '16px',
              lineHeight: 1.7,
              maxWidth: '820px',
            }}>
              中小学课程大纲的统一维护入口。
              职业教育和成人教育后续使用独立的“教学依据”语义，
              不直接复用K12教材版本和册次规则。
            </div>

            <CourseOutlinesPage embedded />
          </div>
        )}

        {sub === 'domains' && (
          <OrganizationEducationDomainsPanel />
        )}
      </div>
    </div>
  )
}
