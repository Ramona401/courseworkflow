/**
 * LibraryPage — 共享教案市场
 *
 * 页面只负责筛选与布局；请求、互动和域切换失效由数据Hook统一处理。
 */

import {
  useEffect,
  useState,
  type CSSProperties,
} from 'react'
import { useNavigate } from 'react-router-dom'
import { useSubjects } from '@/hooks/useSubjects'
import {
  useSharedLessonPlanLibrary,
  type LibraryScope,
} from './useSharedLessonPlanLibrary'
import {
  C,
  EmptyState,
  FilterSelect,
  GRADES,
  LibraryCard,
  SCOPE_TABS,
  SkeletonCard,
} from './LibraryComponents'

export default function LibraryPage() {
  const navigate = useNavigate()

  const [scope, setScope] =
    useState<LibraryScope>('group')
  const [keyword, setKeyword] = useState('')
  const [subject, setSubject] = useState('全部')
  const [grade, setGrade] = useState('全部')
  const [qualityLevel, setQualityLevel] =
    useState('全部')
  const [structureType, setStructureType] =
    useState('全部')

  const {
    subjects,
    loading: subjectsLoading,
  } = useSubjects({
    withAll: true,
  })

  useEffect(() => {
    if (
      !subjectsLoading &&
      !subjects.includes(subject)
    ) {
      setSubject('全部')
    }
  }, [
    subjectsLoading,
    subjects,
    subject,
  ])

  const data = useSharedLessonPlanLibrary({
    scope,
    keyword,
    subject,
    grade,
    qualityLevel,
    structureType,
  })

  const resetFilters = () => {
    setKeyword('')
    setSubject('全部')
    setGrade('全部')
    setQualityLevel('全部')
    setStructureType('全部')
  }

  const filtered =
    keyword.trim() !== '' ||
    subject !== '全部' ||
    grade !== '全部' ||
    qualityLevel !== '全部' ||
    structureType !== '全部'

  return (
    <div>
      <header style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        marginBottom: 20,
      }}>
        <p style={{
          margin: 0,
          color: C.textSec,
          fontSize: 14,
        }}>
          浏览共享教案，Fork优秀教案到我的草稿进行微调
        </p>
        <button
          onClick={() => navigate('/lesson-plans')}
          style={primaryButtonStyle}
        >
          ✨ 新建教案
        </button>
      </header>

      <nav style={{
        display: 'flex',
        gap: 4,
        borderBottom: `1px solid ${C.border}`,
        marginBottom: 20,
      }}>
        {SCOPE_TABS.map(tab => {
          const active = tab.key === scope
          return (
            <button
              key={tab.key}
              title={tab.desc}
              onClick={() => {
                setScope(tab.key)
                resetFilters()
              }}
              style={{
                padding: '12px 20px',
                border: 'none',
                borderBottom: active
                  ? `2px solid ${C.primary}`
                  : '2px solid transparent',
                background: 'transparent',
                color: active
                  ? C.primary
                  : C.textSec,
                fontWeight: active ? 600 : 400,
                cursor: 'pointer',
              }}
            >
              {tab.icon} {tab.label}
              {active &&
                !data.loading &&
                data.total > 0 &&
                ` (${data.total})`}
            </button>
          )
        })}
      </nav>

      <section style={{
        display: 'flex',
        alignItems: 'center',
        gap: 16,
        flexWrap: 'wrap',
        padding: '16px 20px',
        marginBottom: 20,
        background: C.card,
        border: `1px solid ${C.border}`,
        borderRadius: 12,
      }}>
        <input
          value={keyword}
          onChange={event =>
            setKeyword(event.target.value)
          }
          placeholder="搜索标题、课题、学科..."
          style={{
            flex: '1 1 220px',
            minWidth: 180,
            padding: '7px 12px',
            borderRadius: 8,
            border: `1px solid ${
              keyword ? C.primary : C.border
            }`,
          }}
        />

        <FilterSelect
          label="学科"
          value={subject}
          options={subjects}
          onChange={setSubject}
        />
        <FilterSelect
          label="年级"
          value={grade}
          options={GRADES}
          onChange={setGrade}
        />
        <FilterSelect
          label="质量"
          value={qualityLevel}
          options={['全部', '5', '4', '3', '2']}
          onChange={setQualityLevel}
        />
        <FilterSelect
          label="教法"
          value={structureType}
          options={['全部', '1', '2', '3', '4', '5']}
          onChange={setStructureType}
        />

        {filtered && (
          <button
            onClick={resetFilters}
            style={textButtonStyle}
          >
            清空筛选
          </button>
        )}

        {!data.loading && (
          <span style={{
            marginLeft: 'auto',
            color: C.textMuted,
            fontSize: 13,
          }}>
            共 {data.total} 份
          </span>
        )}
      </section>

      {data.error && (
        <div style={errorBoxStyle}>
          <span>⚠️ {data.error}</span>
          <button
            onClick={() => void data.reload()}
            style={textButtonStyle}
          >
            重试
          </button>
        </div>
      )}

      {scope === 'group' &&
        !data.loading &&
        data.myGroups.length === 0 &&
        !data.error && (
          <div style={warningBoxStyle}>
            💡 你还没有加入任何教研组，加入后可查看组内共享教案。
          </div>
        )}

      <main style={{
        display: 'grid',
        gridTemplateColumns:
          'repeat(auto-fill,minmax(340px,1fr))',
        gap: 16,
      }}>
        {data.loading &&
          Array.from({ length: 6 }).map((_, index) => (
            <SkeletonCard key={index} />
          ))}

        {!data.loading &&
          data.plans.map(plan => (
            <LibraryCard
              key={plan.id}
              plan={plan}
              currentUserId={data.userId}
              forkingId={data.forkingId}
              interactions={
                data.interactionsMap[plan.id]
              }
              likePending={Boolean(
                data.interactionPending[
                  `${plan.id}:like`
                ],
              )}
              favoritePending={Boolean(
                data.interactionPending[
                  `${plan.id}:favorite`
                ],
              )}
              forkDisabled={Boolean(data.forkingId)}
              onFork={data.forkPlan}
              onToggleInteraction={
                data.toggleInteraction
              }
            />
          ))}

        {!data.loading &&
          !data.error &&
          data.plans.length === 0 && (
            <EmptyState
              filtered={filtered}
              scope={scope}
              onReset={resetFilters}
            />
          )}
      </main>

      {data.toast && (
        <div style={{
          position: 'fixed',
          bottom: 32,
          left: '50%',
          transform: 'translateX(-50%)',
          padding: '12px 24px',
          borderRadius: 10,
          background: data.toast.type === 'error'
            ? '#FEF2F2'
            : '#1F2937',
          color: data.toast.type === 'error'
            ? C.danger
            : '#fff',
          boxShadow: '0 8px 24px rgba(0,0,0,.15)',
          zIndex: 9999,
        }}>
          {data.toast.type === 'error' ? '⚠️ ' : '✓ '}
          {data.toast.msg}
        </div>
      )}
    </div>
  )
}

const primaryButtonStyle: CSSProperties = {
  padding: '9px 18px',
  borderRadius: 8,
  border: 'none',
  background: C.primary,
  color: '#fff',
  fontWeight: 600,
  cursor: 'pointer',
}

const textButtonStyle: CSSProperties = {
  border: 'none',
  background: 'transparent',
  color: C.primary,
  cursor: 'pointer',
}

const errorBoxStyle: CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  padding: '12px 16px',
  marginBottom: 16,
  borderRadius: 8,
  border: '1px solid #FECACA',
  background: '#FEF2F2',
  color: C.danger,
}

const warningBoxStyle: CSSProperties = {
  padding: '18px 22px',
  marginBottom: 16,
  borderRadius: 10,
  border: '1px solid rgba(245,158,11,.2)',
  background: 'rgba(245,158,11,.06)',
  color: C.warning,
}
