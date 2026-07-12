/**
 * class-profiles.ts — 班级学情 API 封装（差异化教学·老师私有资料，独立模块）
 *
 * 对应后端：handlers/class_profile_handler.go · services/class_profile_service.go · class_profile_ai_summary.go
 *
 * 班级卡（批次1）：
 *   GET    /api/v1/class-profiles        列出我的班级学情卡 → {profiles,total}
 *   POST   /api/v1/class-profiles        新建一张卡（手写入口）→ {profile}
 *   GET    /api/v1/class-profiles/{id}   卡片详情（含四大段）→ {profile}
 *   PUT    /api/v1/class-profiles/{id}   更新卡片 → {updated:true}
 *   DELETE /api/v1/class-profiles/{id}   软删除
 *
 * 学生个体档案（批次2a，挂在班级卡 {id} 下）：
 *   GET    /api/v1/class-profiles/{id}/students         列出该班学生 → {students,total}
 *   POST   /api/v1/class-profiles/{id}/students         新建学生（学号留空自动编号）→ {student}
 *   PUT    /api/v1/class-profiles/{id}/students/{sid}   更新学生 → {student}
 *   DELETE /api/v1/class-profiles/{id}/students/{sid}   删除学生 → {deleted:true}
 *
 * 成绩单导入（批次2b / 2b-2，挂在学生集合下的固定子路径 import）：
 *   POST   /api/v1/class-profiles/{id}/students/import  导入成绩 → {result}
 *
 * AI 总结学情（批次2c，挂在学生集合下的固定子路径 summarize）：
 *   POST   /api/v1/class-profiles/{id}/students/summarize  → {result:ClassSummaryResult}
 *     ⚠ 只生成不落库：后端把学生明细就地脱敏聚合成匿名统计量喂 AI，返回四大段供前端预览，
 *       老师点"采用"后才用这四段调 updateClassProfile 写回班级卡（复用现成更新通道）。
 *     合规：学生个体明细（含学号代号）永不进 AI；进 AI 的只有匿名群体统计量。
 *
 * 三层数据结构：
 *   1. 学生个体档案（本地明细，永不注入 AI）—— 批次2a（录入）+ 批次2b（成绩导入）
 *   2. AI 总结学情 —— 批次2c
 *   3. 班级学情卡（群体结论，注入 AI）—— 批次1
 *
 * 拦截器已处理 code!==0 抛错，本文件直接取 data.data。
 */
import apiClient from './client'

// 学生分层标签（v1 固定 ABC 三层；空串=未分层）
export type StudentTier = '' | 'A' | 'B' | 'C'

// 学情卡最近一次更新来源（仅留痕展示用）
export type ClassAnalyzedFrom = 'manual' | 'ai_chat' | 'score_import' | 'ai_summary'

/** 班级学情卡列表项（不含四大段正文，列表轻量） */
export interface ClassProfileListItem {
  id: string
  subject: string
  grade: string
  class_name: string
  term: string
  student_count: number
  has_profile: boolean
  last_analyzed_at: string | null
  last_analyzed_from: ClassAnalyzedFrom | ''
  updated_at: string
}

/** 班级学情卡详情（含四大段群体学情内容） */
export interface ClassProfileDetail {
  id: string
  scope: string
  scope_target_id: string
  subject: string
  grade: string
  class_name: string
  term: string
  student_count: number
  overall_profile: string
  tier_structure: string
  weak_points: string
  teaching_advice: string
  last_analyzed_at: string | null
  last_analyzed_from: ClassAnalyzedFrom | ''
  created_by: string
  status: 'active' | 'archived'
  created_at: string
  updated_at: string
}

/** 新建班级学情卡请求体（四大段可留空，老师后续慢慢补） */
export interface CreateClassProfileRequest {
  subject: string
  grade: string
  class_name: string
  term?: string
  student_count?: number
  overall_profile?: string
  tier_structure?: string
  weak_points?: string
  teaching_advice?: string
}

/** 更新班级学情卡请求体 */
export interface UpdateClassProfileRequest {
  subject: string
  grade: string
  class_name: string
  term: string
  student_count: number
  overall_profile: string
  tier_structure: string
  weak_points: string
  teaching_advice: string
}

export interface ClassProfileListResponse {
  profiles: ClassProfileListItem[]
  total: number
}

// ==================== 学生个体档案（批次2a）====================

/** 学生历次成绩的一条记录（成绩走批次2b 导入，前端只读展示） */
export interface ClassStudentScore {
  name: string
  score: number
  max: number
  at: string
}

/** 学生档案展示结构（scores 已由后端解析为数组） */
export interface ClassStudentView {
  id: string
  student_code: string
  tier: StudentTier
  scores: ClassStudentScore[]
  latest_score: number | null
  weak_topics: string
  note: string
  updated_at: string
}

/** 新建学生请求体（不含成绩字段——成绩走批次2b 导入） */
export interface CreateClassStudentRequest {
  student_code?: string
  tier?: StudentTier
  weak_topics?: string
  note?: string
}

/** 更新学生请求体（学号必填，同样不含成绩字段） */
export interface UpdateClassStudentRequest {
  student_code: string
  tier: StudentTier
  weak_topics: string
  note: string
}

export interface ClassStudentListResponse {
  students: ClassStudentView[]
  total: number
}

// ==================== 成绩单导入（批次2b / 2b-2）====================

/** 成绩单的一行（2b-2：考试名/日期在请求体顶层；每行带 学号/分数/薄弱点/备注） */
export interface ImportScoreRow {
  student_code: string
  score: number
  weak_topics: string
  note: string
}

/** 成绩单导入请求体（2b-2：考试名/日期整批统一） */
export interface ImportScoresRequest {
  exam_name: string
  exam_date: string
  rows: ImportScoreRow[]
}

/** 成绩单导入结果 */
export interface ImportScoresResult {
  total_rows: number
  appended_scores: number
  affected_students: number
  created_students: number
  profile_updated: number
  skipped_rows: number
  errors: string[]
}

// ==================== AI 总结学情（批次2c）====================

/**
 * AI 总结结果（只生成不落库）。
 * 四大段填进预览弹窗，老师点"采用"后用这四段调 updateClassProfile 写回班级卡。
 * stats_text 是喂给 AI 的匿名统计量原文（已脱敏，可安全展示给老师，增强可信度）。
 */
export interface ClassSummaryResult {
  overall_profile: string
  tier_structure: string
  weak_points: string
  teaching_advice: string
  student_count: number
  stats_text: string
}

// ==================== 班级卡 API（批次1）====================

/** 列出我的全部班级学情卡 */
export async function getClassProfiles(): Promise<ClassProfileListResponse> {
  const { data } = await apiClient.get('/class-profiles')
  return data.data as ClassProfileListResponse
}

/** 取单张班级学情卡详情 */
export async function getClassProfile(id: string): Promise<ClassProfileDetail> {
  const { data } = await apiClient.get(`/class-profiles/${id}`)
  return data.data.profile as ClassProfileDetail
}

/** 新建一张班级学情卡（手写入口） */
export async function createClassProfile(req: CreateClassProfileRequest): Promise<ClassProfileDetail> {
  const { data } = await apiClient.post('/class-profiles', req)
  return data.data.profile as ClassProfileDetail
}

/** 更新班级学情卡 */
export async function updateClassProfile(id: string, req: UpdateClassProfileRequest): Promise<void> {
  await apiClient.put(`/class-profiles/${id}`, req)
}

/** 软删除班级学情卡 */
export async function deleteClassProfile(id: string): Promise<void> {
  await apiClient.delete(`/class-profiles/${id}`)
}

// ==================== 学生档案 API（批次2a）====================

/** 列出某班级的全部学生档案 */
export async function getClassStudents(classProfileId: string): Promise<ClassStudentListResponse> {
  const { data } = await apiClient.get(`/class-profiles/${classProfileId}/students`)
  return data.data as ClassStudentListResponse
}

/** 新建一条学生档案（学号留空则后端自动编号），返回新建后的学生 */
export async function createClassStudent(
  classProfileId: string,
  req: CreateClassStudentRequest,
): Promise<ClassStudentView> {
  const { data } = await apiClient.post(`/class-profiles/${classProfileId}/students`, req)
  return data.data.student as ClassStudentView
}

/** 更新一条学生档案，返回更新后的学生 */
export async function updateClassStudent(
  classProfileId: string,
  studentId: string,
  req: UpdateClassStudentRequest,
): Promise<ClassStudentView> {
  const { data } = await apiClient.put(`/class-profiles/${classProfileId}/students/${studentId}`, req)
  return data.data.student as ClassStudentView
}

/** 删除一条学生档案 */
export async function deleteClassStudent(classProfileId: string, studentId: string): Promise<void> {
  await apiClient.delete(`/class-profiles/${classProfileId}/students/${studentId}`)
}

// ==================== 成绩单导入 API（批次2b / 2b-2）====================

/** 导入一批成绩（2b-2）：考试名/日期顶层，每行带 学号/分数/薄弱点/备注 */
export async function importClassScores(
  classProfileId: string,
  req: ImportScoresRequest,
): Promise<ImportScoresResult> {
  const { data } = await apiClient.post(`/class-profiles/${classProfileId}/students/import`, req)
  return data.data.result as ImportScoresResult
}

// ==================== AI 总结学情 API（批次2c）====================

/**
 * 让 AI 基于该班学生明细的匿名统计量，生成班级卡四大段（只生成不落库）。
 * 老师在预览弹窗点"采用"后，前端用返回的四大段调 updateClassProfile 写回（决策2-B）。
 */
export async function summarizeClassProfile(classProfileId: string): Promise<ClassSummaryResult> {
  const { data } = await apiClient.post(`/class-profiles/${classProfileId}/students/summarize`)
  return data.data.result as ClassSummaryResult
}


// ==================== 按分数线自动分层 API（批次2d）====================

/** 自动分层请求：两条分数线 */
export interface AutoTierRequest {
  a_line: number   // A 层下限：>= 此分 → A
  c_line: number   // C 层上限：< 此分 → C；介于 [c_line, a_line) → B
}

/** 自动分层结果（各层人数 + 跳过统计） */
export interface AutoTierResult {
  total_students: number     // 班级总学生数
  tier_a: number             // 归入 A 层人数
  tier_b: number             // 归入 B 层人数
  tier_c: number             // 归入 C 层人数
  skipped_no_score: number   // 因无成绩被跳过、分层未动的人数
  updated: number            // 实际发生分层变更并写库的人数
}

/**
 * 按两条分数线把本班「有成绩」的学生自动归入 ABC 三层（决策：重算全部有成绩学生，覆盖现有分层）。
 * 无成绩的学生跳过、分层不动。返回各层人数与跳过统计。
 */
export async function autoTierStudents(
  classProfileId: string,
  req: AutoTierRequest,
): Promise<AutoTierResult> {
  const { data } = await apiClient.post(`/class-profiles/${classProfileId}/students/auto-tier`, req)
  return data.data.result as AutoTierResult
}
