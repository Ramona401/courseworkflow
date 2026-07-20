/**
 * RecipeModeSelector — 教育域术语适配的配方三态选择器
 *
 * K12具体年级继续使用历史归一化别名。
 * 非K12学习层级采用原值精确比较，避免被K12年级规则转换。
 */

import type {
  RecipeSelectionMode,
} from '@/api/lesson-plans'
import type {
  RecipeListItem,
} from '@/api/recipes'
import {
  useEducationProfile,
} from '@/hooks/useEducationProfile'
import {
  getEducationLevelLabel,
} from '@/education-domain/options'
import { C } from './workshopConstants'

interface RecipeModeSelectorProps {
  mode: RecipeSelectionMode
  setMode: (
    mode: RecipeSelectionMode,
  ) => void

  recipes: RecipeListItem[]
  recipeId: string
  setRecipeId: (id: string) => void

  currentSubject: string
  currentGrade: string

  loading?: boolean
}

const SCOPE_LABELS: Record<string, string> = {
  school: '学校',
  group: '教研组',
  personal: '个人',
}

const K12_GRADE_ALIASES: Record<string, string> = {
  '1': '1',
  '1年级': '1',
  '一年级': '1',
  '2': '2',
  '2年级': '2',
  '二年级': '2',
  '3': '3',
  '3年级': '3',
  '三年级': '3',
  '4': '4',
  '4年级': '4',
  '四年级': '4',
  '5': '5',
  '5年级': '5',
  '五年级': '5',
  '6': '6',
  '6年级': '6',
  '六年级': '6',
  '7': '7',
  '7年级': '7',
  '七年级': '7',
  '初一': '7',
  '8': '8',
  '8年级': '8',
  '八年级': '8',
  '初二': '8',
  '9': '9',
  '9年级': '9',
  '九年级': '9',
  '初三': '9',
  '10': '10',
  '高一': '10',
  '11': '11',
  '高二': '11',
  '12': '12',
  '高三': '12',
}

function normalizeK12Grade(
  value: string,
): string {
  return K12_GRADE_ALIASES[
    value.trim()
  ] || ''
}

export default function RecipeModeSelector({
  mode,
  setMode,
  recipes,
  recipeId,
  setRecipeId,
  currentSubject,
  currentGrade,
  loading = false,
}: RecipeModeSelectorProps) {
  const {
    domain,
    profile,
    isK12,
  } = useEducationProfile()

  const matches = (
    recipe: RecipeListItem,
  ) => {
    const subjectMatches =
      recipe.subject.trim() ===
      currentSubject.trim()

    const gradeMatches = isK12
      ? Boolean(
          normalizeK12Grade(
            recipe.grade_range,
          ),
        ) &&
        normalizeK12Grade(
          recipe.grade_range,
        ) ===
          normalizeK12Grade(currentGrade)
      : recipe.grade_range.trim() ===
          currentGrade.trim()

    return {
      subjectMatches,
      gradeMatches,
      exact:
        subjectMatches && gradeMatches,
    }
  }

  const selectedRecipe =
    recipes.find(item => item.id === recipeId)

  const selectedMatch =
    selectedRecipe
      ? matches(selectedRecipe)
      : null

  const chooseMode = (
    nextMode: RecipeSelectionMode,
  ) => {
    if (
      nextMode === 'selected' &&
      recipes.length === 0
    ) {
      return
    }

    setMode(nextMode)

    if (nextMode === 'selected') {
      if (!recipeId && recipes.length > 0) {
        setRecipeId(recipes[0].id)
      }
      return
    }

    setRecipeId('')
  }

  const options: {
    mode: RecipeSelectionMode
    title: string
    description: string
    icon: string
  }[] = [
    {
      mode: 'auto',
      title: '智能选择',
      description:
        `只匹配当前${profile.subject_label}和${profile.grade_label}`,
      icon: '✨',
    },
    {
      mode: 'selected',
      title: '指定配方',
      description: '由老师明确选择本次配方',
      icon: '📦',
    },
    {
      mode: 'none',
      title: '不使用',
      description: '只使用系统阶段骨架',
      icon: '○',
    },
  ]

  const currentGradeLabel =
    getEducationLevelLabel(
      domain,
      currentGrade,
    )

  return (
    <div>
      <label style={{
        display: 'block',
        fontSize: '13px',
        fontWeight: 600,
        color: C.textSec,
        marginBottom: '8px',
      }}>
        📦 {profile.lesson_plan_label}配方
      </label>

      <div style={{
        display: 'flex',
        flexWrap: 'wrap',
        gap: '8px',
      }}>
        {options.map(option => {
          const active =
            mode === option.mode

          const disabled =
            option.mode === 'selected' &&
            recipes.length === 0

          return (
            <button
              key={option.mode}
              type="button"
              disabled={disabled}
              onClick={() =>
                chooseMode(option.mode)
              }
              style={{
                flex: '1 1 145px',
                padding: '10px 11px',
                borderRadius: '10px',
                border: `1.5px solid ${
                  active
                    ? C.primary
                    : C.border
                }`,
                background:
                  active
                    ? C.primaryLight
                    : C.card,
                color: disabled
                  ? C.textMuted
                  : active
                    ? C.primary
                    : C.text,
                cursor: disabled
                  ? 'not-allowed'
                  : 'pointer',
                opacity: disabled ? 0.55 : 1,
                textAlign: 'left',
              }}
            >
              <div style={{
                fontSize: '13px',
                fontWeight: 700,
              }}>
                {option.icon}
                {' '}
                {option.title}
              </div>

              <div style={{
                marginTop: '3px',
                fontSize: '10px',
                color: C.textMuted,
                lineHeight: 1.45,
              }}>
                {disabled
                  ? '当前账号暂无可用配方'
                  : option.description}
              </div>
            </button>
          )
        })}
      </div>

      {mode === 'auto' && (
        <div style={{
          marginTop: '7px',
          padding: '7px 10px',
          borderRadius: '8px',
          background: '#F0FDF4',
          color: '#166534',
          fontSize: '11px',
          lineHeight: 1.55,
        }}>
          ✓ 只会自动使用与“
          {currentSubject} ·
          {currentGradeLabel}”
          严格一致的配方；没有命中时使用系统阶段骨架。
        </div>
      )}

      {mode === 'selected' && (
        <div style={{ marginTop: '8px' }}>
          <select
            value={recipeId}
            disabled={
              loading ||
              recipes.length === 0
            }
            onChange={event =>
              setRecipeId(
                event.target.value,
              )
            }
            style={{
              width: '100%',
              padding: '11px 14px',
              borderRadius: '10px',
              border: `1.5px solid ${
                recipeId
                  ? '#F59E0B'
                  : C.border
              }`,
              background: C.card,
              fontSize: '14px',
            }}
          >
            <option value="">
              请选择本次使用的配方
            </option>

            {recipes.map(recipe => {
              const match = matches(recipe)

              return (
                <option
                  key={recipe.id}
                  value={recipe.id}
                >
                  {match.exact
                    ? '【精准】'
                    : '【可选】'}
                  [{SCOPE_LABELS[recipe.scope] || '个人'}]
                  {' '}
                  {recipe.name}
                  {' · '}
                  {recipe.subject || '未标课程'}
                  {' / '}
                  {recipe.grade_range || '未标层级'}
                </option>
              )
            })}
          </select>

          {selectedRecipe &&
           selectedMatch && (
            <div style={{
              marginTop: '6px',
              padding: '8px 10px',
              borderRadius: '8px',
              background:
                selectedMatch.exact
                  ? '#F0FDF4'
                  : '#FFFBEB',
              color:
                selectedMatch.exact
                  ? '#166534'
                  : '#92400E',
              fontSize: '11px',
              lineHeight: 1.6,
            }}>
              {selectedMatch.exact
                ? `✓ 已选择精准匹配的「${selectedRecipe.name}」。`
                : `⚠️ 所选配方标注为“${selectedRecipe.subject || '未标课程'} · ${selectedRecipe.grade_range || '未标层级'}”，与当前课程不完全一致；由于是老师明确选择，平台仍会完整使用。`}
            </div>
          )}
        </div>
      )}

      {mode === 'none' && (
        <div style={{
          marginTop: '7px',
          padding: '7px 10px',
          borderRadius: '8px',
          background: '#F8FAFC',
          color: '#64748B',
          fontSize: '11px',
        }}>
          ✓ 本次不加载配方，只使用系统阶段骨架和已关联资料。
        </div>
      )}
    </div>
  )
}
