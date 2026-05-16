/**
 * tokens.ts — Token积分系统API封装
 *
 * v128 新增（阶段C · Token/积分系统）
 * v129 改造（积分机制融合 · 对齐AOCI精确积分计算）：
 *   - 所有金额类型从整数改为浮点数
 *   - 新增积分策略API（系统策略/学校策略 CRUD）
 *   - 新增模型单价API（CRUD）
 *   - 新增模型积分预览API
 *   - 新增模拟计算API
 *   - ConsumptionListItem 新增9个精确计算字段
 */
import apiClient from './client'

// ==================== 类型定义 ====================

/** 积分账户列表项 */
export interface TokenAccountListItem {
  id: string
  account_type: string
  account_type_name: string
  owner_id: string
  display_name: string
  balance: number
  frozen_amount: number
  available_balance: number
  total_consumed: number
  total_quota: number
  monthly_quota: number
  usage_percent: number
  status: string
  status_name: string
  child_count: number
  expires_at: string | null
  created_at: string
}

/** 积分账户详情 */
export interface TokenAccountDetail {
  id: string
  account_type: string
  account_type_name: string
  owner_id: string
  display_name: string
  balance: number
  frozen_amount: number
  available_balance: number
  total_consumed: number
  total_quota: number
  monthly_quota: number
  usage_percent: number
  status: string
  status_name: string
  expires_at: string | null
  alert_config: TokenAlertConfig | null
  child_accounts: TokenAccountListItem[]
}

/** 预警配置 */
export interface TokenAlertConfig {
  id: string
  account_id: string
  warn_threshold: number
  urgent_threshold: number
  is_enabled: boolean
  last_warn_at: string | null
  last_urgent_at: string | null
}

/** 分配记录列表项 */
export interface AllocationListItem {
  id: string
  from_account_name: string
  to_account_name: string
  amount: number
  allocation_type: string
  memo: string
  operator_name: string
  created_at: string
}

/** 消费流水列表项（v129新增精确字段） */
export interface ConsumptionListItem {
  id: string
  account_name: string
  user_name: string
  amount: number
  balance_before: number
  balance_after: number
  scene_code: string
  model_used: string
  tokens_used: number
  memo: string
  created_at: string
  // v129新增：精确积分计算字段
  input_tokens: number
  output_tokens: number
  model_name: string
  provider: string
  cost_usd: number
  exchange_rate: number
  multiplier: number
  credits_consumed: number
  latency_ms: number
}

/** 采购记录列表项 */
export interface PurchaseListItem {
  id: string
  account_name: string
  amount: number
  purchase_type: string
  order_no: string
  memo: string
  operator_name: string
  valid_until: string | null
  created_at: string
}

/** 概览统计 */
export interface TokenOverviewStats {
  total_accounts: number
  total_balance: number
  total_consumed: number
  total_quota: number
  today_consumed: number
  month_consumed: number
  low_balance_count: number
  expiring_soon_count: number
}

/** 我的积分响应 */
export interface MyTokenAccountResponse {
  has_account: boolean
  message?: string
  account?: {
    id: string
    balance: number
    frozen_amount: number
    total_consumed: number
    total_quota: number
    monthly_quota: number
    status: string
  }
  available_balance?: number
}

// ==================== v129新增：积分策略类型 ====================

/** 积分策略 */
export interface CreditPolicy {
  id: string
  scope: 'system' | 'school'
  scope_id: string | null
  exchange_rate: number
  multiplier: number
  description: string
  updated_by: string | null
  created_at: string
  updated_at: string
}

/** 策略列表项（含学校名称） */
export interface CreditPolicyListItem extends CreditPolicy {
  effective_rate: number
  school_name: string
}

/** 模型单价 */
export interface ModelPrice {
  id: string
  model_name: string
  provider: string
  cost_per_1k_input: number
  cost_per_1k_output: number
  display_name: string
  is_active: boolean
  created_at: string
  updated_at: string
}

/** 模型积分预览 */
export interface ModelPricePreview {
  model_name: string
  provider: string
  display_name: string
  cost_per_1k_input: number
  cost_per_1k_output: number
  credits_per_1k_input: number
  credits_per_1k_output: number
}

/** 积分计算结果（模拟用） */
export interface CreditCalculation {
  input_tokens: number
  output_tokens: number
  model_name: string
  provider: string
  cost_usd: number
  exchange_rate: number
  multiplier: number
  credits_consumed: number
}

// ==================== 请求类型 ====================

export interface CreateAccountRequest {
  account_type: string
  owner_id: string
  parent_account_id?: string
  display_name: string
  monthly_quota?: number
}

export interface AllocateTokensRequest {
  to_account_id: string
  amount: number
  memo?: string
}

export interface PurchaseTokensRequest {
  account_id: string
  amount: number
  purchase_type: string
  order_no?: string
  memo?: string
  valid_until?: string
}

export interface UpdateAlertConfigRequest {
  warn_threshold: number
  urgent_threshold: number
  is_enabled: boolean
}

export interface UpdateCreditPolicyRequest {
  exchange_rate?: number
  multiplier?: number
  description?: string
}

export interface CreateModelPriceRequest {
  model_name: string
  provider: string
  cost_per_1k_input: number
  cost_per_1k_output: number
  display_name: string
}

export interface UpdateModelPriceRequest {
  cost_per_1k_input?: number
  cost_per_1k_output?: number
  display_name?: string
  is_active?: boolean
}

export interface SimulateCreditRequest {
  model_name: string
  input_tokens: number
  output_tokens: number
  school_id?: string
}

// ==================== 辅助函数 ====================

function extractData<T>(resp: { data?: { data?: T } }): T {
  const d = resp?.data as Record<string, unknown> | undefined
  if (d && 'data' in d) return d.data as T
  return d as unknown as T
}

// ==================== 原有API函数 ====================

/** 获取我的积分账户 */
export async function getMyTokenAccount() {
  const resp = await apiClient.get('/tokens/my-account')
  return extractData<MyTokenAccountResponse>(resp)
}

/** 获取概览统计 */
export async function getTokenOverview() {
  const resp = await apiClient.get('/tokens/overview')
  return extractData<TokenOverviewStats>(resp)
}

/** 获取账户列表 */
export async function getTokenAccounts(params?: {
  type?: string; parent_id?: string; status?: string; limit?: number; offset?: number
}) {
  const resp = await apiClient.get('/tokens/accounts', { params })
  return extractData<{ items: TokenAccountListItem[]; total: number }>(resp)
}

/** 获取账户详情 */
export async function getTokenAccountDetail(id: string) {
  const resp = await apiClient.get(`/tokens/accounts/${id}`)
  return extractData<TokenAccountDetail>(resp)
}

/** 创建账户 */
export async function createTokenAccount(req: CreateAccountRequest) {
  const resp = await apiClient.post('/tokens/accounts', req)
  return extractData<TokenAccountListItem>(resp)
}

/** 更新账户状态 */
export async function updateTokenAccountStatus(id: string, status: string) {
  const resp = await apiClient.put(`/tokens/accounts/${id}/status`, { status })
  return extractData<{ message: string }>(resp)
}

/** 分配积分 */
export async function allocateTokens(fromAccountId: string, req: AllocateTokensRequest) {
  const resp = await apiClient.post(`/tokens/accounts/${fromAccountId}/allocate`, req)
  return extractData<{ message: string }>(resp)
}

/** 获取分配记录 */
export async function getTokenAllocations(params?: {
  from_account_id?: string; to_account_id?: string; limit?: number; offset?: number
}) {
  const resp = await apiClient.get('/tokens/allocations', { params })
  return extractData<{ items: AllocationListItem[]; total: number }>(resp)
}

/** 采购/充值 */
export async function purchaseTokens(req: PurchaseTokensRequest) {
  const resp = await apiClient.post('/tokens/purchases', req)
  return extractData<{ message: string }>(resp)
}

/** 获取采购记录 */
export async function getTokenPurchases(params?: {
  account_id?: string; limit?: number; offset?: number
}) {
  const resp = await apiClient.get('/tokens/purchases', { params })
  return extractData<{ items: PurchaseListItem[]; total: number }>(resp)
}

/** 获取消费流水 */
export async function getTokenConsumption(params?: {
  account_id?: string; user_id?: string; scene_code?: string; limit?: number; offset?: number
}) {
  const resp = await apiClient.get('/tokens/consumption', { params })
  return extractData<{ items: ConsumptionListItem[]; total: number }>(resp)
}

/** 获取预警配置 */
export async function getAlertConfig(accountId: string) {
  const resp = await apiClient.get(`/tokens/accounts/${accountId}/alert-config`)
  return extractData<TokenAlertConfig | null>(resp)
}

/** 更新预警配置 */
export async function updateAlertConfig(accountId: string, req: UpdateAlertConfigRequest) {
  const resp = await apiClient.put(`/tokens/accounts/${accountId}/alert-config`, req)
  return extractData<{ message: string }>(resp)
}

// ==================== v129新增：积分策略API ====================

/** 获取策略列表 */
export async function getCreditPolicies() {
  const resp = await apiClient.get('/tokens/credit-policies')
  return extractData<CreditPolicyListItem[]>(resp)
}

/** 获取系统策略 */
export async function getSystemCreditPolicy() {
  const resp = await apiClient.get('/tokens/credit-policies/system')
  return extractData<CreditPolicy>(resp)
}

/** 更新系统策略 */
export async function updateSystemCreditPolicy(req: UpdateCreditPolicyRequest) {
  const resp = await apiClient.put('/tokens/credit-policies/system', req)
  return extractData<CreditPolicy>(resp)
}

/** 获取学校策略 */
export async function getSchoolCreditPolicy(schoolId: string) {
  const resp = await apiClient.get(`/tokens/credit-policies/school/${schoolId}`)
  return extractData<CreditPolicy | { using_system_default: boolean; message: string }>(resp)
}

/** 更新学校策略 */
export async function updateSchoolCreditPolicy(schoolId: string, req: UpdateCreditPolicyRequest) {
  const resp = await apiClient.put(`/tokens/credit-policies/school/${schoolId}`, req)
  return extractData<CreditPolicy>(resp)
}

/** 删除学校策略 */
export async function deleteSchoolCreditPolicy(schoolId: string) {
  const resp = await apiClient.delete(`/tokens/credit-policies/school/${schoolId}`)
  return extractData<{ message: string }>(resp)
}

// ==================== v129新增：模型单价API ====================

/** 获取模型单价列表 */
export async function getModelPrices(includeInactive?: boolean) {
  const resp = await apiClient.get('/tokens/model-prices', { params: includeInactive ? { include_inactive: 'true' } : {} })
  return extractData<ModelPrice[]>(resp)
}

/** 创建模型单价 */
export async function createModelPrice(req: CreateModelPriceRequest) {
  const resp = await apiClient.post('/tokens/model-prices', req)
  return extractData<ModelPrice>(resp)
}

/** 更新模型单价 */
export async function updateModelPrice(id: string, req: UpdateModelPriceRequest) {
  const resp = await apiClient.put(`/tokens/model-prices/${id}`, req)
  return extractData<ModelPrice>(resp)
}

/** 删除模型单价 */
export async function deleteModelPrice(id: string) {
  const resp = await apiClient.delete(`/tokens/model-prices/${id}`)
  return extractData<{ message: string }>(resp)
}

/** 获取模型积分预览 */
export async function getModelPreviews() {
  const resp = await apiClient.get('/tokens/model-previews')
  return extractData<ModelPricePreview[]>(resp)
}

/** 模拟积分计算 */
export async function simulateCredits(req: SimulateCreditRequest) {
  const resp = await apiClient.post('/tokens/simulate', req)
  return extractData<CreditCalculation>(resp)
}

// ==================== 常量 ====================

/** 账户类型选项 */
export const ACCOUNT_TYPE_OPTIONS = [
  { value: 'region', label: '区域账户' },
  { value: 'school', label: '学校账户' },
  { value: 'personal', label: '个人账户' },
]

/** 采购类型选项 */
export const PURCHASE_TYPE_OPTIONS = [
  { value: 'purchase', label: '采购' },
  { value: 'recharge', label: '充值' },
  { value: 'gift', label: '赠送' },
  { value: 'system', label: '系统补贴' },
]

/** 账户状态配色 */
export const ACCOUNT_STATUS_COLORS: Record<string, string> = {
  active: '#10B981',
  suspended: '#EF4444',
  expired: '#9CA3AF',
}

/** AI场景中文名 */
export const SCENE_CODE_LABELS: Record<string, string> = {
  scanner: '课程扫描',
  evaluator: '课件评估',
  meta: '元评估',
  translator: '翻译方案',
  reviewer: '方案审核',
  generator: '页面生成',
  generator_create: '页面创建',
  generator_merge: '页面合并',
  ai_fix: 'AI快修',
  lesson_plan: '备课对话',
  stage_coach: '教练评估',
  assistant_designer: '助手创作',
}

/** 供应商配色 */
export const PROVIDER_COLORS: Record<string, string> = {
  anthropic: '#D97706',
  google: '#4285F4',
  openai: '#10A37F',
  unknown: '#9CA3AF',
}
