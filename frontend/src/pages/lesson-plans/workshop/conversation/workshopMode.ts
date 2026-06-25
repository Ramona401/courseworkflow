/**
 * workshopMode.ts — 备课工坊「对话模式/专家模式」偏好管理（迭代3.5 Phase A）
 *
 * 模式决策优先级（resolveWorkshopMode）：
 *   1. URL 参数 ?mode=expert|conversation（培训现场可强制覆盖）
 *   2. 当前活跃教案的教案级记忆（localStorage workshop_plan_mode_{planId}）
 *      —— 实现设计文档 2.7「存量教案恢复时沿用其创建时模式」
 *   3. 用户全局偏好（localStorage workshop_mode）
 *   4. 系统默认值 DEFAULT_WORKSHOP_MODE
 *
 * 【全局回退开关】出问题时把 DEFAULT_WORKSHOP_MODE 改为 'expert' 即可
 * 让所有未显式选择过模式的用户回到旧版阶段式界面（设计文档第八章回滚方案）。
 *
 * 所有 localStorage 访问都包 try-catch：隐私模式/无痕窗口下 localStorage
 * 可能抛异常，偏好读写失败时静默降级到默认值，绝不阻塞备课主流程。
 */

/** 工坊模式类型：conversation=对话模式（新） | expert=专家模式（原阶段式界面） */
export type WorkshopMode = 'conversation' | 'expert'

/** 系统默认模式 —— 全局回退只改这一行 */
export const DEFAULT_WORKSHOP_MODE: WorkshopMode = 'conversation'

/** 全局偏好的 localStorage 键 */
const GLOBAL_MODE_KEY = 'workshop_mode'

/** 教案级模式记忆的 localStorage 键（按教案ID区分） */
const planModeKey = (planId: string) => `workshop_plan_mode_${planId}`

/** 校验字符串是否为合法模式值 */
function isValidMode(v: string | null | undefined): v is WorkshopMode {
  return v === 'conversation' || v === 'expert'
}

/**
 * 解析当前应使用的工坊模式
 * @param urlMode URL 上的 ?mode= 参数值（可为 null）
 */
export function resolveWorkshopMode(urlMode?: string | null): WorkshopMode {
  // 第1优先级：URL 强制覆盖
  if (isValidMode(urlMode)) return urlMode

  try {
    // 第2优先级：当前活跃教案的教案级记忆（与两模式共享同一个会话键）
    const activePlanId = sessionStorage.getItem('workshop_active_plan_id')
    if (activePlanId) {
      const planMode = localStorage.getItem(planModeKey(activePlanId))
      if (isValidMode(planMode)) return planMode
    }
    // 第3优先级：用户全局偏好
    const globalMode = localStorage.getItem(GLOBAL_MODE_KEY)
    if (isValidMode(globalMode)) return globalMode
  } catch {
    // localStorage/sessionStorage 不可用时静默落到默认值
  }

  // 第4优先级：系统默认
  return DEFAULT_WORKSHOP_MODE
}

/**
 * 持久化模式切换：写全局偏好 + （若有活跃教案）写教案级记忆
 * 由模式切换按钮调用
 */
export function persistWorkshopMode(mode: WorkshopMode): void {
  try {
    localStorage.setItem(GLOBAL_MODE_KEY, mode)
    const activePlanId = sessionStorage.getItem('workshop_active_plan_id')
    if (activePlanId) {
      localStorage.setItem(planModeKey(activePlanId), mode)
    }
  } catch {
    // 写入失败不影响本次会话内的内存态切换
  }
}

/**
 * 记录某教案的创建/使用模式（开始备课、恢复备课时调用）
 * 让该教案下次恢复时沿用本模式
 */
export function recordPlanMode(planId: string, mode: WorkshopMode): void {
  try {
    localStorage.setItem(planModeKey(planId), mode)
  } catch {
    // 静默忽略
  }
}
