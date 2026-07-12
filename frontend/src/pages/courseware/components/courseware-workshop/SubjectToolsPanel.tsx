/**
 * SubjectToolsPanel.tsx — Step5「🧪学科工具」聚合面板
 *
 * 第5批B优化：
 *   1. 统一所有学科工具弹窗外壳尺寸与视觉质感；
 *   2. 入口卡片分组展示，降低“工具堆叠感”；
 *   3. 只改聚合入口，不重写数学/分子/力学大文件，降低风险；
 *   4. 物理实验/化学实验仍使用第5批A的底部课堂控制条。
 *
 * 第33批地理扩展：
 *   1. 增加“地理工具”独立分组；
 *   2. 接入地理互动实验室入口卡片；
 *   3. 接入GeographyLabModal；
 *   4. 地理模板支持AI新建、改编与融入课件。
 */

import { lazy, Suspense, useState } from 'react'
import { refinePage } from '@/api/coursewares'
import { C } from './workshopConstants'

/**
 * 学科工具弹窗二级按需加载。
 *
 * SubjectToolsPanel只负责展示轻量工具宫格。
 * 老师点击某个工具后，浏览器才下载对应弹窗及其模板实现。
 * 这样打开“学科工具”Tab时，不再一次性加载全部学科资源。
 */
const StrokeOrderModal = lazy(
  () => import('./StrokeOrderModal'),
)

const FormulaEditorModal = lazy(
  () => import('./FormulaEditorModal'),
)

const MusicScoreModal = lazy(
  () => import('./MusicScoreModal'),
)

const MathGraphModal = lazy(
  () => import('./MathGraphModal'),
)

const GeographyLabModal = lazy(
  () => import('./GeographyLabModal'),
)

const MoleculeLabModal = lazy(
  () => import('./MoleculeLabModal'),
)

const LifeScienceLabModal = lazy(
  () => import('./LifeScienceLabModal'),
)

const ChemExperimentModal = lazy(
  () => import('./ChemExperimentModal'),
)

const PhysicsLabModal = lazy(
  () => import('./PhysicsLabModal'),
)

const PhysicsSceneModal = lazy(
  () => import('./PhysicsSceneModal'),
)

const ImmersiveLifeScienceModal = lazy(
  () => import('./ImmersiveLifeScienceModal'),
)

// ==================== 类型 ====================

interface Props {
  /** 课件ID */
  coursewareId: string
  /** 当前选中页号 */
  pageNum: number
  /** 页面HTML更新回调 */
  onPageUpdated?: (
    pageNum: number,
    html: string,
  ) => void
}

type ToolKey =
  | 'stroke'
  | 'formula'
  | 'music'
  | 'math'
  | 'geography'
  | 'molecule'
  | 'chemexp'
  | 'physicslab'
  | 'physics'
  | 'lifescience'
  | 'immersive-lifescience'

interface ToolCard {
  key: ToolKey
  emoji: string
  name: string
  desc: string
  color: string
  group:
    | '基础表达'
    | '数学图形'
    | '地理工具'
    | '生命科学'
    | '化学工具'
    | '物理工具'
  badge: string
  available: boolean
}

// ==================== 统一弹窗外壳样式 ====================
// 说明：各弹窗大多用内联style写width/height。
// 这里用!important统一覆盖。
// 好处：不用重写MathGraphModal、MoleculeLabModal、
// PhysicsSceneModal等大文件。

const GLOBAL_MODAL_POLISH_CSS = `
.subject-tools-panel-scope div[style*="z-index: 99993"] {
  backdrop-filter: blur(5px) !important;
  background: rgba(8, 18, 32, 0.62) !important;
}

.subject-tools-panel-scope div[style*="z-index: 99993"] > div {
  width: min(1520px, 98vw) !important;
  height: min(900px, 96vh) !important;
  border-radius: 24px !important;
  box-shadow: 0 34px 88px rgba(0,0,0,0.38) !important;
}

.subject-tools-panel-scope div[style*="z-index: 99993"] > div > div:first-child {
  padding: 16px 24px !important;
}

.subject-tools-panel-scope div[style*="z-index: 99993"] iframe,
.subject-tools-panel-scope div[style*="z-index: 99993"] canvas {
  image-rendering: auto;
}

.subject-tools-panel-scope .st-card {
  transition:
    transform 0.16s ease,
    box-shadow 0.16s ease,
    border-color 0.16s ease;
}

.subject-tools-panel-scope .st-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 30px rgba(15, 23, 42, 0.10) !important;
}

.subject-tools-panel-scope .st-card-shine {
  position: absolute;
  inset: 0;
  opacity: 0;
  background:
    radial-gradient(
      circle at 16% 10%,
      rgba(255,255,255,0.95),
      rgba(255,255,255,0) 36%
    );
  transition: opacity 0.16s ease;
  pointer-events: none;
}

.subject-tools-panel-scope .st-card:hover .st-card-shine {
  opacity: 1;
}
`

// ==================== 工具注册表 ====================

const TOOLS: ToolCard[] = [
  {
    key: 'stroke',
    emoji: '✍️',
    name: '笔顺动画',
    desc: '汉字笔顺演示与描红练习，适合语文识字与书写课。',
    color: '#F59E0B',
    group: '基础表达',
    badge: '语文',
    available: true,
  },
  {
    key: 'formula',
    emoji: '📐',
    name: '公式编辑器',
    desc: 'LaTeX数理化公式，实时预览，适合严谨表达。',
    color: '#1E40AF',
    group: '基础表达',
    badge: '公式',
    available: true,
  },
  {
    key: 'music',
    emoji: '🎼',
    name: '五线谱编辑器',
    desc: 'ABC记谱法编辑乐谱，可试听可播放。',
    color: '#B45309',
    group: '基础表达',
    badge: '音乐',
    available: true,
  },
  {
    key: 'math',
    emoji: '📊',
    name: '数学动态图形',
    desc: '函数图像、几何作图、AI改编，融入后课堂仍可拖动。',
    color: '#7C3AED',
    group: '数学图形',
    badge: '交互图形',
    available: true,
  },
  {
    key: 'geography',
    emoji: '🌍',
    name: '地理互动实验室',
    desc: '经纬网、等高线、地球运动等互动探究，支持AI改编。',
    color: '#0F766E',
    group: '地理工具',
    badge: '空间探究',
    available: true,
  },
  {
    key: 'lifescience',
    emoji: '🧬',
    name: '生命科学实验室',
    desc: '显微镜、细胞结构与生命过程互动观察，支持AI改编。',
    color: '#059669',
    group: '生命科学',
    badge: '结构观察',
    available: true,
  },
  {
    key: 'molecule',
    emoji: '⚗️',
    name: '分子实验室',
    desc: '3D分子可旋转观察，2D结构式支持SMILES。',
    color: '#059669',
    group: '化学工具',
    badge: '结构观察',
    available: true,
  },
  {
    key: 'chemexp',
    emoji: '🧪',
    name: '化学实验',
    desc: '过滤、结晶、气体制取、酸碱中和等实验过程演示。',
    color: '#047857',
    group: '化学工具',
    badge: '实验过程',
    available: true,
  },
  {
    key: 'physicslab',
    emoji: '🔭',
    name: '物理实验室',
    desc: '电路、光学、波动、电磁感应等互动实验。',
    color: '#0284C7',
    group: '物理工具',
    badge: '非力学实验',
    available: true,
  },
  {
    key: 'physics',
    emoji: '🎯',
    name: '力学场景',
    desc: '自由落体、碰撞、单摆、热运动等物理引擎仿真。',
    color: '#DC2626',
    group: '物理工具',
    badge: '力学仿真',
    available: true,
  },
  {
    key: 'immersive-lifescience',
    emoji: '🌐',
    name: '3D生命科学实验室',
    desc: '完整3D学习工作台，整页应用到课件，不是小插件。',
    color: '#047857',
    group: '生命科学',
    badge: '整页3D',
    available: true,
  },
]

const GROUPS: ToolCard['group'][] = [
  '基础表达',
  '数学图形',
  '地理工具',
  '生命科学',
  '化学工具',
  '物理工具',
]

const GROUP_META: Record<
  ToolCard['group'],
  {
    emoji: string
    desc: string
    bg: string
    border: string
  }
> = {
  基础表达: {
    emoji: '🧩',
    desc: '文字、公式、乐谱等通用学科表达组件',
    bg: 'linear-gradient(135deg,#FFFBEB,#FFFFFF)',
    border: '#FDE68A',
  },
  数学图形: {
    emoji: '📊',
    desc: '函数、几何、动态交互与AI改编',
    bg: 'linear-gradient(135deg,#F5F3FF,#FFFFFF)',
    border: '#DDD6FE',
  },
  地理工具: {
    emoji: '🌍',
    desc: '空间定位、地形判读与地球运动互动探究',
    bg: 'linear-gradient(135deg,#F0FDFA,#EFF6FF)',
    border: '#99F6E4',
  },
  生命科学: {
    emoji: '🧬',
    desc: '显微观察、细胞结构与生命过程互动模型',
    bg: 'linear-gradient(135deg,#ECFDF5,#FFFFFF)',
    border: '#A7F3D0',
  },
  化学工具: {
    emoji: '⚗️',
    desc: '分子结构观察与实验过程演示',
    bg: 'linear-gradient(135deg,#ECFDF5,#FFFFFF)',
    border: '#BBF7D0',
  },
  物理工具: {
    emoji: '🔭',
    desc: '力学仿真与电学、光学、波动实验',
    bg: 'linear-gradient(135deg,#EFF6FF,#FFFFFF)',
    border: '#BAE6FD',
  },
}

const TOOL_NOUN: Record<ToolKey, string> = {
  stroke: '笔顺动画',
  formula: '公式',
  music: '五线谱',
  math: '数学动态图形',
  geography: '地理互动组件',
  molecule: '分子模型',
  lifescience: '生命科学组件',
  chemexp: '化学实验',
  physicslab: '物理实验',
  physics: '力学场景',
  'immersive-lifescience': '3D生命科学实验',
}

/**
 * 单个学科工具动态资源加载期间的局部占位。
 *
 * 使用固定全屏遮罩，让老师点击卡片后立即获得明确反馈；
 * 仅遮罩当前页面，不触发App层全局路由加载占位。
 */
function SubjectToolLoadingFallback() {
  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 99993,
        background: 'rgba(8,18,32,0.62)',
        backdropFilter: 'blur(5px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <div
        style={{
          minWidth: 260,
          padding: '28px 34px',
          borderRadius: 20,
          background: '#fff',
          boxShadow:
            '0 28px 80px rgba(0,0,0,0.36)',
          textAlign: 'center',
        }}
      >
        <div
          style={{
            fontSize: 34,
            marginBottom: 12,
          }}
        >
          🧪
        </div>

        <div
          style={{
            fontSize: 15,
            fontWeight: 800,
            color: '#1F2937',
          }}
        >
          学科工具加载中...
        </div>

        <div
          style={{
            marginTop: 7,
            fontSize: 12.5,
            color: '#6B7280',
          }}
        >
          首次打开该工具需要加载互动资源
        </div>
      </div>
    </div>
  )
}

// ==================== 组件 ====================

export default function SubjectToolsPanel({
  coursewareId,
  pageNum,
  onPageUpdated,
}: Props) {
  const [activeModal, setActiveModal] =
    useState<ToolKey | null>(null)

  const [inserting, setInserting] =
    useState(false)

  const [message, setMessage] =
    useState('')

  const makeInsertHandler =
    (tool: ToolKey) =>
      async (instruction: string) => {
        if (
          !coursewareId ||
          pageNum <= 0 ||
          inserting
        ) {
          return
        }

        setInserting(true)
        setMessage('')

        try {
          const result = await refinePage(
            coursewareId,
            pageNum,
            instruction,
          )

          if (
            result.html_content &&
            onPageUpdated
          ) {
            onPageUpdated(
              pageNum,
              result.html_content,
            )
          }

          setMessage(
            '✅ ' +
            TOOL_NOUN[tool] +
            '已融入第 ' +
            pageNum +
            ' 页！请在上方预览区查看效果',
          )

          setActiveModal(null)
        } catch (error) {
          setMessage(
            '❌ 融入失败: ' +
            (
              error instanceof Error
                ? error.message
                : '未知错误'
            ),
          )
        } finally {
          setInserting(false)
        }
      }

  const handleCardClick = (
    tool: ToolCard,
  ) => {
    if (!tool.available) return

    if (pageNum <= 0) {
      setMessage(
        '❌ 请先在上方预览区选择要融入的页面',
      )

      return
    }

    setMessage('')
    setActiveModal(tool.key)
  }

  return (
    <Suspense fallback={<SubjectToolLoadingFallback />}>
    <div
      className="subject-tools-panel-scope"
      style={{ marginTop: 16 }}
    >
      <style>{GLOBAL_MODAL_POLISH_CSS}</style>

      <div
        style={{
          padding: 20,
          borderRadius: 18,
          border: '1px solid #E5E7EB',
          background:
            'linear-gradient(180deg,#FFFFFF 0%,#FAFBFF 100%)',
          boxShadow:
            '0 8px 26px rgba(15,23,42,0.04)',
        }}
      >
        {/* 顶部说明 */}
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: 14,
            marginBottom: 18,
          }}
        >
          <div
            style={{
              width: 44,
              height: 44,
              borderRadius: 14,
              background:
                'linear-gradient(135deg,#EEF2FF,#E0F2FE)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 23,
              flexShrink: 0,
              boxShadow:
                '0 6px 16px rgba(2,132,199,0.12)',
            }}
          >
            🧪
          </div>

          <div
            style={{
              minWidth: 0,
              flex: 1,
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                flexWrap: 'wrap',
              }}
            >
              <div
                style={{
                  fontSize: 17,
                  fontWeight: 850,
                  color: C.textPrimary,
                }}
              >
                学科工具
              </div>

              <span
                style={{
                  padding: '5px 13px',
                  borderRadius: 999,
                  background: C.primaryBg,
                  color: C.primary,
                  fontSize: 13,
                  fontWeight: 750,
                  whiteSpace: 'nowrap',
                }}
              >
                将融入：第 {pageNum || '—'} 页
              </span>
            </div>

            <div
              style={{
                fontSize: 13,
                color: C.textSecondary,
                lineHeight: 1.65,
                marginTop: 6,
              }}
            >
              选择工具后进入大弹窗编辑。
              右侧面板用于设置初始状态，
              真正进入课件的是中间互动组件；
              地理、生命科学、物理和化学组件融入后会保留课堂控制条。
            </div>
          </div>
        </div>

        {/* 分组工具 */}
        <div
          style={{
            display: 'grid',
            gap: 16,
          }}
        >
          {GROUPS.map(group => {
            const meta =
              GROUP_META[group]

            const items =
              TOOLS.filter(
                tool => tool.group === group,
              )

            return (
              <section
                key={group}
                style={{
                  border:
                    '1px solid ' +
                    meta.border,
                  borderRadius: 16,
                  background: meta.bg,
                  padding: 14,
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 9,
                    marginBottom: 12,
                  }}
                >
                  <span
                    style={{ fontSize: 18 }}
                  >
                    {meta.emoji}
                  </span>

                  <span
                    style={{
                      fontSize: 14,
                      fontWeight: 850,
                      color: C.textPrimary,
                    }}
                  >
                    {group}
                  </span>

                  <span
                    style={{
                      fontSize: 12,
                      color: C.textMuted,
                    }}
                  >
                    {meta.desc}
                  </span>
                </div>

                <div
                  style={{
                    display: 'grid',
                    gridTemplateColumns:
                      'repeat(auto-fill,minmax(250px,1fr))',
                    gap: 12,
                  }}
                >
                  {items.map(tool => (
                    <div
                      key={tool.key}
                      className="st-card"
                      onClick={() =>
                        handleCardClick(tool)
                      }
                      style={{
                        padding: '15px 16px',
                        borderRadius: 15,
                        background: '#fff',
                        border:
                          '1.5px solid #E5E7EB',
                        cursor:
                          tool.available
                            ? 'pointer'
                            : 'default',
                        opacity:
                          tool.available
                            ? 1
                            : 0.65,
                        position: 'relative',
                        overflow: 'hidden',
                        boxShadow:
                          '0 3px 12px rgba(15,23,42,0.045)',
                      }}
                      onMouseEnter={event => {
                        if (tool.available) {
                          event.currentTarget.style.borderColor =
                            tool.color
                        }
                      }}
                      onMouseLeave={event => {
                        if (tool.available) {
                          event.currentTarget.style.borderColor =
                            '#E5E7EB'
                        }
                      }}
                    >
                      <div className="st-card-shine" />

                      <div
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 11,
                          marginBottom: 8,
                          position: 'relative',
                        }}
                      >
                        <span
                          style={{
                            width: 40,
                            height: 40,
                            borderRadius: 13,
                            background:
                              'linear-gradient(135deg,#F8FAFC,#FFFFFF)',
                            border:
                              '1px solid #EEF2F7',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent:
                              'center',
                            fontSize: 22,
                            flexShrink: 0,
                          }}
                        >
                          {tool.emoji}
                        </span>

                        <div
                          style={{ minWidth: 0 }}
                        >
                          <div
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 8,
                              flexWrap: 'wrap',
                            }}
                          >
                            <span
                              style={{
                                fontSize: 15,
                                fontWeight: 850,
                                color:
                                  tool.color,
                              }}
                            >
                              {tool.name}
                            </span>

                            <span
                              style={{
                                padding:
                                  '2px 8px',
                                borderRadius: 999,
                                background:
                                  tool.color +
                                  '14',
                                color:
                                  tool.color,
                                fontSize: 10.5,
                                fontWeight: 800,
                              }}
                            >
                              {tool.badge}
                            </span>
                          </div>
                        </div>
                      </div>

                      <div
                        style={{
                          fontSize: 12.2,
                          color:
                            C.textSecondary,
                          lineHeight: 1.58,
                          minHeight: 38,
                          position: 'relative',
                        }}
                      >
                        {tool.desc}
                      </div>

                      {tool.available && (
                        <div
                          style={{
                            marginTop: 10,
                            fontSize: 12.2,
                            fontWeight: 800,
                            color: tool.color,
                            position:
                              'relative',
                          }}
                        >
                          打开编辑器 →
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </section>
            )
          })}
        </div>

        {message && (
          <div
            style={{
              marginTop: 14,
              padding: '11px 14px',
              borderRadius: 10,
              background:
                message.startsWith('❌')
                  ? '#FEE2E2'
                  : '#D1FAE5',
              color:
                message.startsWith('❌')
                  ? '#DC2626'
                  : '#059669',
              fontSize: 13,
              fontWeight: 650,
            }}
          >
            {message}
          </div>
        )}

        {/* 弹窗挂载区 */}
        {activeModal === 'stroke' && (
          <StrokeOrderModal
            pageNum={pageNum}
            inserting={inserting}
            onClose={() =>
              setActiveModal(null)
            }
            onInsert={
              makeInsertHandler('stroke')
            }
          />
        )}

        {activeModal === 'formula' && (
          <FormulaEditorModal
            pageNum={pageNum}
            inserting={inserting}
            onClose={() =>
              setActiveModal(null)
            }
            onInsert={
              makeInsertHandler('formula')
            }
          />
        )}

        {activeModal === 'music' && (
          <MusicScoreModal
            pageNum={pageNum}
            inserting={inserting}
            onClose={() =>
              setActiveModal(null)
            }
            onInsert={
              makeInsertHandler('music')
            }
          />
        )}

        {activeModal === 'math' && (
          <MathGraphModal
            pageNum={pageNum}
            inserting={inserting}
            onClose={() =>
              setActiveModal(null)
            }
            onInsert={
              makeInsertHandler('math')
            }
          />
        )}

        {activeModal === 'geography' && (
          <GeographyLabModal
            pageNum={pageNum}
            inserting={inserting}
            onClose={() =>
              setActiveModal(null)
            }
            onInsert={
              makeInsertHandler('geography')
            }
          />
        )}

        {activeModal === 'molecule' && (
          <MoleculeLabModal
            pageNum={pageNum}
            inserting={inserting}
            onClose={() =>
              setActiveModal(null)
            }
            onInsert={
              makeInsertHandler('molecule')
            }
          />
        )}

        {activeModal === 'lifescience' && (
          <LifeScienceLabModal
            pageNum={pageNum}
            inserting={inserting}
            onClose={() =>
              setActiveModal(null)
            }
            onInsert={
              makeInsertHandler('lifescience')
            }
          />
        )}

        {activeModal === 'chemexp' && (
          <ChemExperimentModal
            pageNum={pageNum}
            inserting={inserting}
            onClose={() =>
              setActiveModal(null)
            }
            onInsert={
              makeInsertHandler('chemexp')
            }
          />
        )}

        {activeModal === 'physicslab' && (
          <PhysicsLabModal
            pageNum={pageNum}
            inserting={inserting}
            onClose={() =>
              setActiveModal(null)
            }
            onInsert={
              makeInsertHandler('physicslab')
            }
          />
        )}

        {activeModal === 'physics' && (
          <PhysicsSceneModal
            pageNum={pageNum}
            inserting={inserting}
            onClose={() =>
              setActiveModal(null)
            }
            onInsert={
              makeInsertHandler('physics')
            }
          />
        )}
      </div>

      {activeModal ===
        'immersive-lifescience' && (
        <ImmersiveLifeScienceModal
          coursewareId={coursewareId}
          pageNum={pageNum}
          onPageUpdated={onPageUpdated}
          onClose={() =>
            setActiveModal(null)
          }
        />
      )}
    </div>
    </Suspense>
  )
}
