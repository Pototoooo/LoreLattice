import { get, post } from '@/utils/request'

export type BillingFeatureKey =
  | 'lorelattice_llm_tokens'
  | 'lorelattice_embedding_tokens'
  | 'lorelattice_rerank_tokens'
  | 'lorelattice_asr_seconds'

export type BillingMode = 'platform' | 'byok' | 'local' | 'included' | 'mixed'

export interface BillingOverview {
  enabled: boolean
  status: string
  financial_status: 'local'
  cny_per_usd: number
  payment_enabled: boolean
  top_up_amounts_fen: number[]
  ai_credits: {
    granted_usd: number
    used_usd: number
    remaining_usd?: number
    currency: 'USD'
  }
}

export interface BillingUsageRow {
  id: string
  feature_key: BillingFeatureKey
  model: string
  provider: string
  operation: string
  job_id: string
  business_category: string
  billing_mode: BillingMode
  chargeable: boolean
  price_version: string
  quantity?: number
  unit: string
  cost_usd?: number
  estimated: boolean
  status: string
  created_at: string
}

export interface BillingUsageJob {
  id: string
  business_category: string
  billing_mode: BillingMode
  call_count: number
  chargeable_calls: number
  cost_usd: number
  estimated: boolean
  status: string
  models: string[]
  providers: string[]
  started_at: string
  completed_at?: string
}

interface Response<T> {
  success: boolean
  data: T
}

export const getBillingOverview = () =>
  get<Response<BillingOverview>>('/api/v1/billing/overview')

export const getBillingUsage = (limit = 50) =>
  get<Response<BillingUsageRow[]>>(`/api/v1/billing/usage?limit=${limit}`)

export const getBillingUsageJobs = (limit = 50) =>
  get<Response<BillingUsageJob[]>>(`/api/v1/billing/usage?view=jobs&limit=${limit}`)

export interface PaymentOrder {
  id: string
  amount_fen: number
  currency: 'CNY'
  status: 'pending' | 'paid' | 'closed'
  expires_at: string
  created_at: string
  paid_at?: string
  checkout_url?: string
}

export const createPayment = (amountFen: number, idempotencyKey: string) =>
  post<Response<PaymentOrder>>('/api/v1/billing/payments', { amount_fen: amountFen, idempotency_key: idempotencyKey })
export const getPaymentOrders = () => get<Response<PaymentOrder[]>>('/api/v1/billing/payments')
export const getPaymentOrder = (id: string) => get<Response<PaymentOrder>>(`/api/v1/billing/payments/${encodeURIComponent(id)}`)
