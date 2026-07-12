import apiClient from './client';

// ==================== 类型定义 ====================

/** 回收站条目（教案/课件通用） */
export interface TrashItem {
  id: string;
  title: string;
  type: 'lesson_plan' | 'courseware';
  subject: string;
  grade: string;
  deleted_at: string;
  created_at: string;
  days_left: number;
}

/** 回收站列表响应 */
export interface TrashListResponse {
  lesson_plans: TrashItem[];
  coursewares: TrashItem[];
  total: number;
}

// ==================== API 函数 ====================

/** 获取回收站列表（教案+课件合并） */
export async function listTrash(): Promise<TrashListResponse> {
  const resp = await apiClient.get('/trash');
  return resp.data.data;
}

/** 恢复回收站中的项目 */
export async function restoreTrashItem(id: string, type: 'lesson_plan' | 'courseware'): Promise<void> {
  await apiClient.post(`/trash/${id}/restore`, { type });
}

/** 永久删除回收站中的项目 */
export async function permanentDeleteTrashItem(id: string, type: 'lesson_plan' | 'courseware'): Promise<void> {
  await apiClient.delete(`/trash/${id}/permanent`, { data: { type } });
}
