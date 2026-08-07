/**
 * lesson-plan-publish-guard.ts — 教案个人发布的浏览器端确认守卫。
 *
 * 浏览器端职责：
 *   1. 记录页面最后一次真正读取到的正式正文快照；
 *   2. 发布前重新读取服务器当前教案，但不覆盖旧确认快照；
 *   3. 页面快照与服务器版本或正文不一致时拒绝发布；
 *   4. 最新聊天若是未提交的完整改稿且与正式正文不同，拒绝发布；
 *   5. 最终把已确认版本提交给后端事务级发布守卫。
 *
 * 后端仍是最终安全边界：作者、状态、版本及Word同步状态会在
 * 同一数据库事务中重新校验。
 */

import apiClient from './client'
import type {
  ConversationMessage,
  LessonPlan,
} from './lesson-plans.types'

interface LessonPlanPublishSnapshot {
  version: number
  contentMarkdown: string
}

const publishSnapshots =
  new Map<
    string,
    LessonPlanPublishSnapshot
  >()

/** 记录页面已经取得并展示的正式教案快照。 */
export function rememberLessonPlanPublishSnapshot(
  plan: LessonPlan,
): LessonPlan {
  if (
    plan &&
    plan.id &&
    Number.isInteger(plan.version) &&
    plan.version > 0
  ) {
    publishSnapshots.set(
      plan.id,
      {
        version: plan.version,
        contentMarkdown:
          plan.content_markdown || '',
      },
    )
  }

  return plan
}

/** 发布个人教案，同时执行浏览器快照交叉校验。 */
export async function publishLessonPlanPersonalGuarded(
  lessonPlanID: string,
): Promise<void> {
  const snapshot =
    publishSnapshots.get(lessonPlanID)

  if (!snapshot) {
    const latest =
      await loadLatestLessonPlan(
        lessonPlanID,
      )

    rememberLessonPlanPublishSnapshot(
      latest,
    )

    throw new Error(
      `已加载服务器最新教案v${latest.version}，请先核对右侧正文，再次点击发布`,
    )
  }

  const [
    latestPlan,
    latestMessages,
  ] = await Promise.all([
    loadLatestLessonPlan(
      lessonPlanID,
    ),
    loadLatestConversationMessages(
      lessonPlanID,
    ),
  ])

  const latestContent =
    latestPlan.content_markdown || ''

  if (
    latestPlan.version !==
      snapshot.version ||
    latestContent !==
      snapshot.contentMarkdown
  ) {
    rememberLessonPlanPublishSnapshot(
      latestPlan,
    )

    throw new Error(
      `教案已经更新为v${latestPlan.version}，当前页面尚未确认这份最新正文；请刷新或重新进入后核对右侧画布`,
    )
  }

  const latestAssistant =
    findLatestAssistantMessage(
      latestMessages,
    )

  if (
    latestAssistant &&
    isUncommittedFormalDraft(
      latestAssistant,
      latestContent,
    )
  ) {
    throw new Error(
      '聊天区最新完整改稿没有成功写入右侧正文和原格式Word，不能按该未保存稿发布；请重新修改并等待右侧出现“已保存”提示',
    )
  }

  await apiClient.post(
    `/lesson-plans/plans/${lessonPlanID}/publish-personal`,
    {
      expected_version:
        snapshot.version,
    },
  )
}

async function loadLatestLessonPlan(
  lessonPlanID: string,
): Promise<LessonPlan> {
  const response =
    await apiClient.get(
      `/lesson-plans/plans/${lessonPlanID}`,
    )

  return response.data.data as LessonPlan
}

async function loadLatestConversationMessages(
  lessonPlanID: string,
): Promise<ConversationMessage[]> {
  const response =
    await apiClient.get(
      `/lesson-plans/plans/${lessonPlanID}/conversation`,
    )

  const messages =
    response.data.data?.messages

  return Array.isArray(messages)
    ? messages
    : []
}

function findLatestAssistantMessage(
  messages: ConversationMessage[],
): ConversationMessage | null {
  for (
    let index = messages.length - 1;
    index >= 0;
    index -= 1
  ) {
    if (
      messages[index]?.role ===
      'assistant'
    ) {
      return messages[index]
    }
  }

  return null
}

function isUncommittedFormalDraft(
  message: ConversationMessage,
  formalContent: string,
): boolean {
  if (
    !isLikelyFullLessonPlan(
      message.content || '',
    )
  ) {
    return false
  }

  if (
    message.metadata?.content_committed ===
    true
  ) {
    return false
  }

  return (
    normalizeContent(message.content) !==
    normalizeContent(formalContent)
  )
}

function normalizeContent(
  value: string,
): string {
  return (value || '')
    .replace(/\r\n/g, '\n')
    .trim()
}

/** 使用与对话页面相同的严格完整教案判据。 */
function isLikelyFullLessonPlan(
  content: string,
): boolean {
  const hasProcess =
    content.includes('教学过程') ||
    content.includes('教学环节') ||
    content.includes('教学活动') ||
    content.includes('环节一') ||
    (
      (
        content.includes('教师话术') ||
        content.includes('教师活动')
      ) &&
      content.includes('学生活动')
    )

  const hasEnding =
    content.includes('板书设计') ||
    content.includes('作业') ||
    content.includes('课堂小结') ||
    content.includes('课堂总结')

  const hasHead =
    content.includes('教学目标') ||
    content.includes('教学重难点') ||
    content.includes('教学重点')

  const hasDetail =
    content.includes('教师话术') ||
    content.includes('学生活动') ||
    content.includes('教师活动') ||
    content.includes('预期反应')

  return (
    hasProcess &&
    hasEnding &&
    hasHead &&
    hasDetail
  )
}
