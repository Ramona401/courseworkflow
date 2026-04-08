/**
 * SceneConfigPanel — 场景配置Tab面板
 *
 * v89-3拆分：从AICenterPage.tsx中拆出场景配置Tab的完整内容
 * 包含：场景分组展示、场景编辑表单、降级模型配置
 */
import type { SceneConfig, UpdateSceneConfigRequest } from '@/api/ai-config'
import { C, ModelSelect } from './AICenterConstants'
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
  onSceneFormChange: (updater: (prev: UpdateSceneConfigRequest) => UpdateSceneConfigRequest) => void
}

export default function SceneConfigPanel({
  scenes, editingScene, sceneForm, sceneSaving,
  availableModels, modelsQueried,
  onEditScene, onCancelEdit, onSaveScene, onSceneFormChange,
}: SceneConfigPanelProps) {
  const lessonPlanScenes = scenes.filter(s => s.scene_group === 'lesson_plan')
  const pipelineScenes = scenes.filter(s => s.scene_group !== 'lesson_plan')

  // 渲染单个场景行
  const renderSceneRow = (scene: SceneConfig) => {
    const isEditing = editingScene === scene.scene_code
    const fbCount = (scene.fallback_models || []).length
    return (
      <div key={scene.scene_code} style={{
        padding: '16px 20px', borderRadius: '12px',
        border: `1px solid ${isEditing ? C.primary : C.border}`,
        background: isEditing ? C.primaryLight : C.bg,
        transition: 'all 200ms ease',
      }}>
        {/* 场景头部 */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <div style={{
              width: '8px', height: '8px', borderRadius: '50%',
              background: scene.is_active ? C.success : C.textMuted,
            }} />
            <span style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>
              {scene.scene_name}
            </span>
            <span style={{
              fontSize: '12px', color: C.textMuted, fontFamily: 'monospace',
              padding: '2px 8px', background: 'rgba(0,0,0,0.05)', borderRadius: '4px',
            }}>
              {scene.scene_code}
            </span>
          </div>
          {!isEditing && (
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <span style={{
                fontSize: '12px', fontFamily: 'monospace',
                color: scene.model && availableModels.includes(scene.model) ? C.success : C.textMuted,
              }}>
                {scene.model || '继承全局'} · T:{scene.temperature ?? '继承'} · Max:{scene.max_tokens ?? '继承'}
                {fbCount > 0 && (
                  <span style={{
                    marginLeft: '6px', padding: '1px 6px', borderRadius: '8px',
                    background: C.warningLight, color: C.warning,
                    fontSize: '11px', fontWeight: 600, fontFamily: 'sans-serif',
                  }}>
                    {fbCount}个降级
                  </span>
                )}
              </span>
              <button
                onClick={() => onEditScene(scene)}
                style={{
                  padding: '5px 14px', borderRadius: '8px',
                  border: `1px solid ${C.border}`, background: C.white,
                  fontSize: '12px', fontWeight: 500, color: C.primary, cursor: 'pointer',
                }}
              >
                编辑
              </button>
            </div>
          )}
        </div>

        {/* 编辑表单 */}
        {isEditing && (
          <div style={{ marginTop: '16px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 120px 140px 100px', gap: '12px', marginBottom: '14px' }}>
              <div>
                <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '4px' }}>
                  主模型
                  {availableModels.length > 0 && (
                    <span style={{ fontWeight: 400, color: C.success, marginLeft: '4px' }}>
                      ({availableModels.length}个可选)
                    </span>
                  )}
                </label>
                <ModelSelect
                  value={sceneForm.model ?? null}
                  onChange={v => onSceneFormChange(p => ({ ...p, model: v }))}
                  availableModels={availableModels}
                  placeholder="输入模型名称"
                />
              </div>
              <div>
                <label style={{ display: 'block', fontSize: '12px', color: C.textSec, marginBottom: '4px' }}>温度</label>
                <input
                  type="number" step="0.1" min="0" max="2"
                  value={sceneForm.temperature ?? ''}
                  onChange={e => onSceneFormChange(p => ({ ...p, temperature: e.target.value === '' ? null : parseFloat(e.target.value) }))}
                  placeholder="继承全局"
                  style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', boxSizing: 'border-box' }}
                  onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                  onBlur={e => { e.currentTarget.style.borderColor = C.border }}
                />
              </div>
              <div>
                <label style={{ display: 'block', fontSize: '12px', color: C.textSec, marginBottom: '4px' }}>Max Tokens</label>
                <input
                  type="number" step="1000"
                  value={sceneForm.max_tokens ?? ''}
                  onChange={e => onSceneFormChange(p => ({ ...p, max_tokens: e.target.value === '' ? null : parseInt(e.target.value) }))}
                  placeholder="继承全局"
                  style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', boxSizing: 'border-box' }}
                  onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                  onBlur={e => { e.currentTarget.style.borderColor = C.border }}
                />
              </div>
              <div>
                <label style={{ display: 'block', fontSize: '12px', color: C.textSec, marginBottom: '4px' }}>状态</label>
                <select
                  value={sceneForm.is_active ? 'true' : 'false'}
                  onChange={e => onSceneFormChange(p => ({ ...p, is_active: e.target.value === 'true' }))}
                  style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', boxSizing: 'border-box', background: C.white }}
                >
                  <option value="true">启用</option>
                  <option value="false">禁用</option>
                </select>
              </div>
            </div>

            {/* 降级模型选择器 */}
            <div style={{ marginBottom: '14px' }}>
              <FallbackModelsPicker
                value={sceneForm.fallback_models || []}
                onChange={v => onSceneFormChange(p => ({ ...p, fallback_models: v }))}
                availableModels={availableModels}
                primaryModel={sceneForm.model ?? null}
              />
            </div>

            <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
              <button
                onClick={onCancelEdit}
                style={{ padding: '7px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '13px', color: C.textSec, cursor: 'pointer' }}
              >
                取消
              </button>
              <button
                onClick={() => onSaveScene(scene.scene_code)}
                disabled={sceneSaving}
                style={{
                  padding: '7px 20px', borderRadius: '8px', border: 'none',
                  background: `linear-gradient(135deg,${C.primary},#7C3AED)`,
                  color: '#fff', fontSize: '13px', fontWeight: 600,
                  cursor: 'pointer', opacity: sceneSaving ? 0.6 : 1,
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

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
      {/* 教案备课场景 */}
      {lessonPlanScenes.length > 0 && (
        <div style={{
          background: C.card, borderRadius: '16px',
          border: `1px solid ${C.border}`,
          boxShadow: '0 2px 8px rgba(0,0,0,0.04)',
        }}>
          <div style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}` }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <span style={{ fontSize: '16px' }}>📝</span>
                  <span style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>教案备课场景配置</span>
                  <span style={{ fontSize: '12px', color: C.textMuted }}>（{lessonPlanScenes.length} 个场景）</span>
                </div>
                <div style={{ fontSize: '13px', color: C.textSec, marginTop: '3px' }}>
                  备课工坊中AI对话、开场白生成、AI评审、建议优化等场景的模型配置
                </div>
              </div>
              {modelsQueried && availableModels.length > 0 && (
                <div style={{
                  padding: '6px 12px', borderRadius: '10px',
                  background: C.successLight, border: '1px solid rgba(16,185,129,0.25)',
                  fontSize: '12px', color: C.success, fontWeight: 500,
                }}>
                  ✓ {availableModels.length} 个模型可选
                </div>
              )}
            </div>
          </div>
          <div style={{ padding: '16px 24px', display: 'flex', flexDirection: 'column', gap: '10px' }}>
            {lessonPlanScenes.map(renderSceneRow)}
          </div>
        </div>
      )}

      {/* Pipeline场景 */}
      <div style={{
        background: C.card, borderRadius: '16px',
        border: `1px solid ${C.border}`,
        boxShadow: '0 2px 8px rgba(0,0,0,0.04)',
      }}>
        <div style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}` }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span style={{ fontSize: '16px' }}>⚙️</span>
                <span style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>Pipeline 场景配置</span>
                <span style={{ fontSize: '12px', color: C.textMuted }}>（{pipelineScenes.length} 个场景）</span>
              </div>
              <div style={{ fontSize: '13px', color: C.textSec, marginTop: '3px' }}>
                课件质量评估Pipeline中各AI步骤的模型配置，每个场景可独立覆盖全局配置
              </div>
            </div>
            {!modelsQueried && (
              <div style={{
                padding: '8px 14px', borderRadius: '10px',
                background: C.warningLight, border: '1px solid rgba(245,158,11,0.25)',
                fontSize: '12px', color: C.warning, fontWeight: 500,
              }}>
                💡 先在"连接配置"查询可用模型
              </div>
            )}
            {modelsQueried && availableModels.length > 0 && (
              <div style={{
                padding: '6px 12px', borderRadius: '10px',
                background: C.successLight, border: '1px solid rgba(16,185,129,0.25)',
                fontSize: '12px', color: C.success, fontWeight: 500,
              }}>
                ✓ {availableModels.length} 个模型可选
              </div>
            )}
          </div>
        </div>
        <div style={{ padding: '16px 24px', display: 'flex', flexDirection: 'column', gap: '10px' }}>
          {pipelineScenes.map(renderSceneRow)}
        </div>
      </div>
    </div>
  )
}
