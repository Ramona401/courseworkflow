/**
 * RecipeModeSelector.tsx — 开始备课时的配方三态选择器
 *
 * 两种备课模式共用同一份交互与教师文案，避免：
 *   - 对话模式把空recipe_id解释为“不使用”；
 *   - 后端却把空recipe_id解释为“自动匹配”；
 *   - 专家模式和对话模式行为不一致。
 *
 * 三态：
 *   auto：平台按照学校、教研组和学科规则自动选择；
 *   selected：老师明确选择一个可用配方；
 *   none：老师明确要求不使用配方，只使用系统阶段骨架。
 */

import type { RecipeSelectionMode } from '@/api/lesson-plans'
import type { RecipeListItem } from '@/api/recipes'
import { C } from './workshopConstants'

interface RecipeModeSelectorProps {
  mode: RecipeSelectionMode
  setMode: (mode: RecipeSelectionMode) => void
  recipes: RecipeListItem[]
  recipeId: string
  setRecipeId: (id: string) => void
  loading?: boolean
}

const RECIPE_SCOPE_LABELS: Record<string, string> = {
  school: '学校',
  group: '教研组',
  personal: '个人',
}

const MODE_OPTIONS: Array<{
  mode: RecipeSelectionMode
  title: string
  description: string
  icon: string
}> = [
  {
    mode: 'auto',
    title: '智能选择',
    description: '平台按学校、教研组和学科规则匹配',
    icon: '✨',
  },
  {
    mode: 'selected',
    title: '指定配方',
    description: '由老师明确选择本次使用的配方',
    icon: '📦',
  },
  {
    mode: 'none',
    title: '不使用',
    description: '只使用系统阶段骨架，不加载配方',
    icon: '○',
  },
]

export default function RecipeModeSelector({
  mode,
  setMode,
  recipes,
  recipeId,
  setRecipeId,
  loading = false,
}: RecipeModeSelectorProps) {
  const selectedRecipe = recipes.find(recipe => recipe.id === recipeId)

  const chooseMode = (nextMode: RecipeSelectionMode) => {
    if (nextMode === 'selected' && recipes.length === 0) return

    setMode(nextMode)

    if (nextMode === 'selected') {
      if (!recipeId && recipes.length > 0) {
        setRecipeId(recipes[0].id)
      }
      return
    }

    // auto和none都不应携带旧的recipe_id。
    // 即使父组件误传，后端也会再次规范化；前端这里先消除歧义。
    setRecipeId('')
  }

  return (
    <div>
      <label
        style={{
          display: 'block',
          fontSize: '13px',
          fontWeight: 600,
          color: C.textSec,
          marginBottom: '8px',
        }}
      >
        📦 备课配方
      </label>

      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: '8px',
        }}
      >
        {MODE_OPTIONS.map(option => {
          const active = mode === option.mode
          const disabled =
            option.mode === 'selected' && recipes.length === 0

          return (
            <button
              key={option.mode}
              type="button"
              disabled={disabled}
              onClick={() => chooseMode(option.mode)}
              style={{
                flex: '1 1 145px',
                minWidth: 0,
                padding: '10px 11px',
                borderRadius: '10px',
                border: `1.5px solid ${
                  active ? C.primary : C.border
                }`,
                background: active ? C.primaryLight : C.card,
                color: disabled
                  ? C.textMuted
                  : active
                    ? C.primary
                    : C.text,
                cursor: disabled ? 'not-allowed' : 'pointer',
                opacity: disabled ? 0.55 : 1,
                textAlign: 'left',
                transition: 'all 150ms ease',
              }}
            >
              <div
                style={{
                  fontSize: '13px',
                  fontWeight: active ? 700 : 600,
                  marginBottom: '3px',
                }}
              >
                {option.icon} {option.title}
                {option.mode === 'auto' && (
                  <span
                    style={{
                      marginLeft: '5px',
                      fontSize: '9px',
                      fontWeight: 600,
                      padding: '1px 5px',
                      borderRadius: '999px',
                      background: '#DCFCE7',
                      color: '#166534',
                    }}
                  >
                    推荐
                  </span>
                )}
              </div>
              <div
                style={{
                  fontSize: '10px',
                  lineHeight: 1.45,
                  color: disabled ? C.textMuted : C.textMuted,
                }}
              >
                {option.mode === 'selected' && recipes.length === 0
                  ? '本学科暂无可用配方'
                  : option.description}
              </div>
            </button>
          )
        })}
      </div>

      {mode === 'auto' && (
        <div
          style={{
            marginTop: '7px',
            padding: '7px 10px',
            borderRadius: '8px',
            background: '#F0FDF4',
            color: '#166534',
            fontSize: '11px',
            lineHeight: 1.55,
          }}
        >
          ✓ 开始后平台会先查学校默认配方，再匹配教研组或学校共享配方；
          每轮回执会告诉你最终是否匹配成功。
        </div>
      )}

      {mode === 'selected' && (
        <div style={{ marginTop: '8px' }}>
          <select
            value={recipeId}
            disabled={loading || recipes.length === 0}
            onChange={event => setRecipeId(event.target.value)}
            style={{
              width: '100%',
              padding: '11px 14px',
              borderRadius: '10px',
              border: `1.5px solid ${
                recipeId ? '#F59E0B' : C.border
              }`,
              fontSize: '14px',
              color: recipeId ? C.text : C.textMuted,
              background: C.card,
              cursor: loading ? 'wait' : 'pointer',
              outline: 'none',
              boxSizing: 'border-box',
            }}
          >
            <option value="">请选择本次使用的配方</option>
            {recipes.map(recipe => (
              <option key={recipe.id} value={recipe.id}>
                [{RECIPE_SCOPE_LABELS[recipe.scope] || '个人'}] {recipe.name}
                {' · '}
                {recipe.component_count}组件
                {' · '}
                用{recipe.use_count}次
              </option>
            ))}
          </select>

          {selectedRecipe && (
            <div
              style={{
                marginTop: '6px',
                padding: '7px 10px',
                borderRadius: '8px',
                background: '#FFFBEB',
                color: '#92400E',
                fontSize: '11px',
                lineHeight: 1.55,
              }}
            >
              ✓ 老师明确选择「{selectedRecipe.name}」。
              {selectedRecipe.description
                ? ` ${selectedRecipe.description}`
                : '本次教案结构、流程和相关教研要求将以此配方为依据。'}
            </div>
          )}
        </div>
      )}

      {mode === 'none' && (
        <div
          style={{
            marginTop: '7px',
            padding: '7px 10px',
            borderRadius: '8px',
            background: '#F8FAFC',
            color: '#64748B',
            fontSize: '11px',
            lineHeight: 1.55,
          }}
        >
          ✓ 已明确不使用配方。平台不会自动匹配配方，
          本次只使用系统阶段骨架和你另外关联的教学资料。
        </div>
      )}
    </div>
  )
}
