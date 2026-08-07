/**
 * SceneConfigPanel — AI场景配置面板
 *
 * 职责：
 *   1. 按教案备课、课件工坊、Pipeline和知识库分组展示场景；
 *   2. 编辑主模型、温度、Max Tokens、启用状态和Fallback模型；
 *   3. 未知分组单独收敛到“其它场景”，避免错误归入Pipeline；
 *   4. 前端只负责展示和提交，场景合法性及运行时模型策略由后端裁决。
 */
import type {
  SceneConfig,
  UpdateSceneConfigRequest,
} from '@/api/ai-config'
import {
  C,
  ModelSelect,
} from './AICenterConstants'
import FallbackModelsPicker from './FallbackModelsPicker'

interface SceneConfigPanelProps {
  scenes: SceneConfig[]
  editingScene: string | null
  sceneForm: UpdateSceneConfigRequest
  sceneSaving: boolean
  availableModels: string[]
  modelsQueried: boolean
  onEditScene: (scene: SceneConfig) => void
  onCancelEdit: () => void
  onSaveScene: (code: string) => void
  onSceneFormChange: (
    updater: (
      prev: UpdateSceneConfigRequest,
    ) => UpdateSceneConfigRequest,
  ) => void
}

interface SceneGroupDefinition {
  key: string
  icon: string
  title: string
  description: string
}

const SCENE_GROUPS: SceneGroupDefinition[] = [
  {
    key: 'lesson_plan',
    icon: '📝',
    title: '教案备课场景配置',
    description:
      '备课工坊中的对话生成、课程大纲Harness、阶段教练和AI助手创作。',
  },
  {
    key: 'courseware',
    icon: '🎞️',
    title: '课件工坊场景配置',
    description:
      '课件方案、HTML生成、页面微调、来源规整、媒体规划和模板处理。',
  },
  {
    key: 'pipeline',
    icon: '⚙️',
    title: 'Pipeline 场景配置',
    description:
      '课程质量评估Pipeline中的扫描、评估、翻译、审核和页面生成步骤。',
  },
  {
    key: 'knowledge_base',
    icon: '📚',
    title: '知识库场景配置',
    description:
      '知识点抽取、课标压缩和多轮结果语义仲裁。',
  },
]

export default function SceneConfigPanel({
  scenes,
  editingScene,
  sceneForm,
  sceneSaving,
  availableModels,
  modelsQueried,
  onEditScene,
  onCancelEdit,
  onSaveScene,
  onSceneFormChange,
}: SceneConfigPanelProps) {
  const knownGroupKeys = new Set(
    SCENE_GROUPS.map(group => group.key),
  )

  const groupedScenes = SCENE_GROUPS
    .map(group => ({
      definition: group,
      items: scenes.filter(
        scene => scene.scene_group === group.key,
      ),
    }))
    .filter(group => group.items.length > 0)

  const unclassifiedScenes = scenes.filter(
    scene => !knownGroupKeys.has(scene.scene_group),
  )

  const renderSceneRow = (
    scene: SceneConfig,
  ) => {
    const isEditing =
      editingScene === scene.scene_code
    const fallbackCount =
      (scene.fallback_models || []).length
    const modelIsKnown =
      Boolean(scene.model) &&
      availableModels.includes(scene.model || '')

    return (
      <div
        key={scene.scene_code}
        style={{
          padding: '16px 20px',
          borderRadius: '12px',
          border: `1px solid ${
            isEditing ? C.primary : C.border
          }`,
          background:
            isEditing ? C.primaryLight : C.bg,
          transition: 'all 200ms ease',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '16px',
          }}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
              minWidth: 0,
            }}
          >
            <div
              style={{
                width: '8px',
                height: '8px',
                borderRadius: '50%',
                flexShrink: 0,
                background:
                  scene.is_active
                    ? C.success
                    : C.textMuted,
              }}
            />
            <span
              style={{
                fontSize: '15px',
                fontWeight: 600,
                color: C.text,
                whiteSpace: 'nowrap',
              }}
            >
              {scene.scene_name}
            </span>
            <span
              style={{
                minWidth: 0,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                fontSize: '12px',
                color: C.textMuted,
                fontFamily: 'monospace',
                padding: '2px 8px',
                background: 'rgba(0,0,0,0.05)',
                borderRadius: '4px',
              }}
              title={scene.scene_code}
            >
              {scene.scene_code}
            </span>
          </div>

          {!isEditing && (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'flex-end',
                gap: '12px',
                flexShrink: 0,
              }}
            >
              <span
                style={{
                  fontSize: '12px',
                  fontFamily: 'monospace',
                  color:
                    modelIsKnown
                      ? C.success
                      : C.textMuted,
                }}
              >
                {scene.model || '继承全局'}
                {' · T:'}
                {scene.temperature ?? '继承'}
                {' · Max:'}
                {scene.max_tokens ?? '继承'}

                {fallbackCount > 0 && (
                  <span
                    style={{
                      marginLeft: '6px',
                      padding: '1px 6px',
                      borderRadius: '8px',
                      background: C.warningLight,
                      color: C.warning,
                      fontSize: '11px',
                      fontWeight: 600,
                      fontFamily: 'sans-serif',
                    }}
                  >
                    {fallbackCount}个降级
                  </span>
                )}
              </span>

              <button
                onClick={() => onEditScene(scene)}
                style={{
                  padding: '5px 14px',
                  borderRadius: '8px',
                  border: `1px solid ${C.border}`,
                  background: C.white,
                  fontSize: '12px',
                  fontWeight: 500,
                  color: C.primary,
                  cursor: 'pointer',
                }}
              >
                编辑
              </button>
            </div>
          )}
        </div>

        {isEditing && (
          <div style={{ marginTop: '16px' }}>
            <div
              style={{
                display: 'grid',
                gridTemplateColumns:
                  'minmax(260px,1fr) 120px 140px 100px',
                gap: '12px',
                marginBottom: '14px',
              }}
            >
              <div>
                <label
                  style={{
                    display: 'block',
                    fontSize: '12px',
                    fontWeight: 600,
                    color: C.textSec,
                    marginBottom: '4px',
                  }}
                >
                  主模型
                  {availableModels.length > 0 && (
                    <span
                      style={{
                        fontWeight: 400,
                        color: C.success,
                        marginLeft: '4px',
                      }}
                    >
                      ({availableModels.length}个可选)
                    </span>
                  )}
                </label>

                <ModelSelect
                  value={sceneForm.model ?? null}
                  onChange={value =>
                    onSceneFormChange(previous => ({
                      ...previous,
                      model: value,
                    }))
                  }
                  availableModels={availableModels}
                  placeholder="输入或选择模型名称"
                />
              </div>

              <div>
                <label
                  style={{
                    display: 'block',
                    fontSize: '12px',
                    color: C.textSec,
                    marginBottom: '4px',
                  }}
                >
                  温度
                </label>
                <input
                  type="number"
                  step="0.1"
                  min="0"
                  max="2"
                  value={sceneForm.temperature ?? ''}
                  onChange={event =>
                    onSceneFormChange(previous => ({
                      ...previous,
                      temperature:
                        event.target.value === ''
                          ? null
                          : Number.parseFloat(
                              event.target.value,
                            ),
                    }))
                  }
                  placeholder="继承全局"
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    borderRadius: '8px',
                    border: `1px solid ${C.border}`,
                    fontSize: '13px',
                    outline: 'none',
                    boxSizing: 'border-box',
                    background: C.white,
                  }}
                  onFocus={event => {
                    event.currentTarget.style.borderColor =
                      C.primary
                  }}
                  onBlur={event => {
                    event.currentTarget.style.borderColor =
                      C.border
                  }}
                />
              </div>

              <div>
                <label
                  style={{
                    display: 'block',
                    fontSize: '12px',
                    color: C.textSec,
                    marginBottom: '4px',
                  }}
                >
                  Max Tokens
                </label>
                <input
                  type="number"
                  step="1000"
                  value={sceneForm.max_tokens ?? ''}
                  onChange={event =>
                    onSceneFormChange(previous => ({
                      ...previous,
                      max_tokens:
                        event.target.value === ''
                          ? null
                          : Number.parseInt(
                              event.target.value,
                              10,
                            ),
                    }))
                  }
                  placeholder="继承全局"
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    borderRadius: '8px',
                    border: `1px solid ${C.border}`,
                    fontSize: '13px',
                    outline: 'none',
                    boxSizing: 'border-box',
                    background: C.white,
                  }}
                  onFocus={event => {
                    event.currentTarget.style.borderColor =
                      C.primary
                  }}
                  onBlur={event => {
                    event.currentTarget.style.borderColor =
                      C.border
                  }}
                />
              </div>

              <div>
                <label
                  style={{
                    display: 'block',
                    fontSize: '12px',
                    color: C.textSec,
                    marginBottom: '4px',
                  }}
                >
                  状态
                </label>
                <select
                  value={
                    sceneForm.is_active
                      ? 'true'
                      : 'false'
                  }
                  onChange={event =>
                    onSceneFormChange(previous => ({
                      ...previous,
                      is_active:
                        event.target.value === 'true',
                    }))
                  }
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    borderRadius: '8px',
                    border: `1px solid ${C.border}`,
                    fontSize: '13px',
                    outline: 'none',
                    boxSizing: 'border-box',
                    background: C.white,
                  }}
                >
                  <option value="true">启用</option>
                  <option value="false">禁用</option>
                </select>
              </div>
            </div>

            <div style={{ marginBottom: '14px' }}>
              <FallbackModelsPicker
                value={
                  sceneForm.fallback_models || []
                }
                onChange={value =>
                  onSceneFormChange(previous => ({
                    ...previous,
                    fallback_models: value,
                  }))
                }
                availableModels={availableModels}
                primaryModel={
                  sceneForm.model ?? null
                }
              />
            </div>

            <div
              style={{
                display: 'flex',
                gap: '8px',
                justifyContent: 'flex-end',
              }}
            >
              <button
                onClick={onCancelEdit}
                style={{
                  padding: '7px 16px',
                  borderRadius: '8px',
                  border: `1px solid ${C.border}`,
                  background: C.white,
                  fontSize: '13px',
                  color: C.textSec,
                  cursor: 'pointer',
                }}
              >
                取消
              </button>

              <button
                onClick={() =>
                  onSaveScene(scene.scene_code)
                }
                disabled={sceneSaving}
                style={{
                  padding: '7px 20px',
                  borderRadius: '8px',
                  border: 'none',
                  background:
                    `linear-gradient(135deg,${C.primary},#7C3AED)`,
                  color: '#fff',
                  fontSize: '13px',
                  fontWeight: 600,
                  cursor:
                    sceneSaving
                      ? 'not-allowed'
                      : 'pointer',
                  opacity: sceneSaving ? 0.6 : 1,
                }}
              >
                {sceneSaving ? '保存中...' : '保存'}
              </button>
            </div>
          </div>
        )}
      </div>
    )
  }

  const renderGroupCard = (
    definition: SceneGroupDefinition,
    items: SceneConfig[],
  ) => (
    <div
      key={definition.key}
      style={{
        background: C.card,
        borderRadius: '16px',
        border: `1px solid ${C.border}`,
        boxShadow: '0 2px 8px rgba(0,0,0,0.04)',
      }}
    >
      <div
        style={{
          padding: '18px 24px',
          borderBottom: `1px solid ${C.border}`,
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '16px',
          }}
        >
          <div>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
              }}
            >
              <span style={{ fontSize: '16px' }}>
                {definition.icon}
              </span>
              <span
                style={{
                  fontSize: '15px',
                  fontWeight: 600,
                  color: C.text,
                }}
              >
                {definition.title}
              </span>
              <span
                style={{
                  fontSize: '12px',
                  color: C.textMuted,
                }}
              >
                （{items.length} 个场景）
              </span>
            </div>
            <div
              style={{
                fontSize: '13px',
                color: C.textSec,
                marginTop: '3px',
              }}
            >
              {definition.description}
            </div>
          </div>

          {!modelsQueried && (
            <div
              style={{
                padding: '8px 14px',
                borderRadius: '10px',
                background: C.warningLight,
                border:
                  '1px solid rgba(245,158,11,0.25)',
                fontSize: '12px',
                color: C.warning,
                fontWeight: 500,
              }}
            >
              💡 先在“连接配置”查询可用模型
            </div>
          )}

          {modelsQueried &&
            availableModels.length > 0 && (
              <div
                style={{
                  padding: '6px 12px',
                  borderRadius: '10px',
                  background: C.successLight,
                  border:
                    '1px solid rgba(16,185,129,0.25)',
                  fontSize: '12px',
                  color: C.success,
                  fontWeight: 500,
                }}
              >
                ✓ {availableModels.length} 个模型可选
              </div>
            )}
        </div>
      </div>

      <div
        style={{
          padding: '16px 24px',
          display: 'flex',
          flexDirection: 'column',
          gap: '10px',
        }}
      >
        {items.map(renderSceneRow)}
      </div>
    </div>
  )

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: '20px',
      }}
    >
      {groupedScenes.map(group =>
        renderGroupCard(
          group.definition,
          group.items,
        ),
      )}

      {unclassifiedScenes.length > 0 &&
        renderGroupCard(
          {
            key: 'other',
            icon: '🧩',
            title: '其它场景配置',
            description:
              '后端尚未归入标准业务分组的兼容场景，请结合场景代码核对用途。',
          },
          unclassifiedScenes,
        )}

      {scenes.length === 0 && (
        <div
          style={{
            padding: '40px 24px',
            textAlign: 'center',
            background: C.card,
            borderRadius: '16px',
            border: `1px solid ${C.border}`,
            color: C.textMuted,
            fontSize: '14px',
          }}
        >
          暂无可展示的AI场景配置
        </div>
      )}
    </div>
  )
}
