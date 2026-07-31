import { get, post } from '@/utils/request'

export type BillingFeatureKey =
  | 'lorelattice_llm_tokens'
  | 'lorelattice_embedding_tokens'
  | 'lorelattice_rerank_tokens'
  | 'lorelattice_asr_seconds'

export interface BillingFeatureOverview {
  key: BillingFeatureKey
  name: string
  unit: 'token' | 'second'
  limit: number
  used: number
  remaining: number
  unit_price_usd: number
}

export interface BillingOverview {
  enabled: boolean
  status: string
  plan_key: 'lorelattice_trial' | 'lorelattice_pro' | ''
  plan_name: string
  trial_ends_at?: string
  period_started_at?: string
  period_ends_at?: string
  features: BillingFeatureOverview[]
  credit_balance_usd?: number
  cancel_scheduled: boolean
}

export interface BillingUsageRow {
  id: string
  feature_key: BillingFeatureKey
  model: string
  provider: string
  operation: string
  quantity?: number
  unit: string
  cost_usd?: number
  estimated: boolean
  status: string
  created_at: string
}

interface Response<T> {
  success: boolean
  data: T
}

export const getBillingOverview = () =>
  get<Response<BillingOverview>>('/api/v1/billing/overview')

export const getBillingUsage = (limit = 50) =>
  get<Response<BillingUsageRow[]>>(`/api/v1/billing/usage?limit=${limit}`)

export const getBillingInvoices = () =>
  get<Response<{ data?: any[]; meta?: any }>>('/api/v1/billing/invoices')

export const createSandboxTopUp = (amount: 1 | 5 | 10, idempotencyKey: string) =>
  post<Response<BillingOverview>>('/api/v1/billing/top-ups', {
    amount,
    idempotency_key: idempotencyKey,
  })

export const upgradeBillingPlan = () =>
  post<Response<BillingOverview>>('/api/v1/billing/subscription/change', {
    plan_key: 'lorelattice_pro',
  })

export const cancelBillingPlan = () =>
  post<Response<BillingOverview>>('/api/v1/billing/subscription/cancel')

export const undoBillingCancellation = () =>
  post<Response<BillingOverview>>('/api/v1/billing/subscription/unschedule-cancel')
