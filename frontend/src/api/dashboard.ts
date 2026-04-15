/**
 * 仪表盘API封装
 * P4.5-D新增：获取仪表盘统计数据
 * FE-CS-01修复：消除 (res.data as any).data 类型绕过，改用 extractData 辅助函数
 */
import client from './client'
import type { ApiResponse } from './client'
import type { AxiosResponse } from 'axios'

// ==================== 辅助函数 ====================

/**
 * extractData 从 Axios 响应中安全提取业务数据
 * FE-CS-01修复：替代 (res.data as any).data 写法
 */
function extractData<T>(res: AxiosResponse<ApiResponse<T>>): T {
  return res.data.data as T
}

// ==================== 类型定义 ====================

/** 仪表盘统计数据 */
export interface DashboardStats {
  /** 课程总数 */
  total_courses: number
  /** 有索引的课程数 */
  courses_with_index: number
  /** Pipeline总数 */
  total_pipelines: number
  /** 运行中Pipeline数 */
  running_pipelines: number
  /** 待审核Pipeline数 */
  review_queue: number
  /** 已定稿Pipeline数 */
  finalized: number
  /** 失败Pipeline数 */
  failed: number
  /** 评估达标Pipeline数（均分≥9.0） */
  passed_count: number
  /** AI总token消耗 */
  total_tokens_used: number
}

// ==================== API方法 ====================

/** 获取仪表盘统计数据 */
export async function getDashboardStats(): Promise<DashboardStats> {
  const res = await client.get<ApiResponse<DashboardStats>>('/dashboard/stats')
  return extractData<DashboardStats>(res)
}
