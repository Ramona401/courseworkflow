/**
 * physicsSceneAI.ts — 力学场景 AI 生成 API
 *
 * 复用 /math-graph/generate，target=physics_scene。
 * 返回 Matter.js setup 构造代码，操作变量 Matter / engine / world / W / H。
 */

import apiClient from './client'

const GENERATE_TIMEOUT_MS = 300000

export interface PhysicsSceneGenerateRequest {
  target: 'physics_scene'
  mode: 'adapt' | 'create'
  description: string
  base_code?: string
  template_name?: string
  image?: string
}

export interface PhysicsSceneGenerateResponse {
  code: string
}

export async function generatePhysicsSceneCode(
  data: PhysicsSceneGenerateRequest,
): Promise<PhysicsSceneGenerateResponse> {
  const resp = await apiClient.post('/math-graph/generate', data, { timeout: GENERATE_TIMEOUT_MS })
  return resp.data.data as PhysicsSceneGenerateResponse
}
