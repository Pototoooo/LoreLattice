<template>
  <div class="billing-page">
    <div class="page-header">
      <div>
        <h2>套餐与用量</h2>
        <p>一个 AI Credits 余额覆盖全部平台模型；BYOK 与本地模型只记录用量，不重复扣费。</p>
      </div>
      <t-button variant="outline" :loading="loading" @click="loadAll">刷新</t-button>
    </div>

    <t-alert v-if="errorMessage" theme="error" :message="errorMessage" />
    <t-alert v-if="overview?.financial_status === 'local_fallback'" theme="warning"
      :message="overview.financial_warning || '远端计费暂时不可用，当前显示本地账本余额。'" />
    <t-alert v-if="partialWarning" theme="warning" :message="partialWarning" />
    <div v-else-if="loading && !overview" class="loading-state">正在读取套餐与用量…</div>

    <template v-if="overview">
      <section class="summary-grid">
        <div class="summary-card">
          <span class="summary-label">当前套餐</span>
          <strong>{{ overview.plan_name || '尚未开通' }}</strong>
          <span class="summary-sub">{{ statusText }}</span>
        </div>
        <div class="summary-card">
          <span class="summary-label">{{ overview.plan_key === 'lorelattice_trial' ? '试用到期' : '本周期结束' }}</span>
          <strong>{{ formatDate(overview.plan_key === 'lorelattice_trial' ? overview.trial_ends_at : overview.period_ends_at) }}</strong>
          <span class="summary-sub">额度在对应周期内有效</span>
        </div>
        <div v-if="isOwner" class="summary-card">
          <span class="summary-label">AI Credits 余额</span>
          <strong>${{ formatMoney(overview.ai_credits.remaining_usd) }}</strong>
          <span class="summary-sub">仅平台托管模型扣减</span>
        </div>
        <div v-else class="summary-card">
          <span class="summary-label">财务操作</span>
          <strong>由 Owner 管理</strong>
          <span class="summary-sub">余额不足时请联系 workspace Owner</span>
        </div>
      </section>

      <section class="panel">
        <div class="panel-title">
          <div>
            <h3>统一 AI Credits</h3>
            <p>五类模型统一结算；Embedding、Rerank 等内部步骤不再显示为独立钱包。</p>
          </div>
        </div>
        <div class="credit-overview">
          <div>
            <span>平台模型已使用</span>
            <strong>${{ formatMoney(overview.ai_credits.used_usd) }}</strong>
          </div>
          <div v-for="mode in overview.billing_modes" :key="mode.mode" class="mode-stat">
            <t-tag size="small" :theme="mode.mode === 'platform' ? 'primary' : 'success'">{{ modeLabel(mode.mode) }}</t-tag>
            <span>{{ mode.calls }} 次调用 · 扣费 ${{ formatMoney(mode.cost_usd) }}</span>
          </div>
        </div>
      </section>

      <section v-if="isOwner" class="panel">
        <div class="panel-title">
          <div>
            <h3>套餐与余额管理</h3>
            <p>当前为 Sandbox 模拟充值，不会发生真实支付。</p>
          </div>
        </div>
        <div class="action-row">
          <div class="action-block">
            <span class="action-label">模拟充值</span>
            <div class="button-group">
              <t-button v-for="amount in topUpAmounts" :key="amount" variant="outline"
                :loading="actionLoading === `topup-${amount}`" @click="topUp(amount)">
                +${{ amount }}
              </t-button>
            </div>
          </div>
          <div class="action-block">
            <span class="action-label">套餐</span>
            <div class="button-group">
              <t-button v-if="overview.plan_key === 'lorelattice_trial'" theme="primary"
                :loading="actionLoading === 'upgrade'" @click="upgrade">立即升级 Pro</t-button>
              <t-button v-else-if="overview.cancel_scheduled" variant="outline"
                :loading="actionLoading === 'undo'" @click="undoCancel">撤销取消</t-button>
              <t-button v-else theme="danger" variant="outline"
                :loading="actionLoading === 'cancel'" @click="cancelPlan">周期末取消</t-button>
            </div>
          </div>
        </div>
      </section>

      <section class="panel">
        <div class="panel-title">
          <div>
            <h3>近期 AI 作业</h3>
            <p>一次提问、一次文档处理或一次转写显示为一条；内部模型调用已自动聚合。</p>
          </div>
        </div>
        <div v-if="jobs.length === 0" class="empty-state">暂无 AI 用量记录</div>
        <div v-else class="table-wrap">
          <table>
            <thead><tr><th>时间</th><th>业务动作</th><th>模型</th><th>调用</th><th>计费方式</th><th>扣费</th><th>状态</th></tr></thead>
            <tbody>
              <tr v-for="job in jobs" :key="job.id">
                <td>{{ formatDate(job.started_at, true) }}</td>
                <td>{{ categoryLabel(job.business_category) }}</td>
                <td>{{ job.models?.join('、') || job.providers?.join('、') || '—' }}</td>
                <td>{{ job.call_count }} 次</td>
                <td><t-tag size="small" :theme="job.billing_mode === 'platform' ? 'primary' : 'success'">{{ modeLabel(job.billing_mode) }}</t-tag></td>
                <td>${{ formatMoney(job.cost_usd) }}</td>
                <td><t-tag size="small" :theme="job.status === 'completed' ? 'success' : 'default'">
                  {{ job.status === 'completed' ? (job.estimated ? '已完成 · 含估算' : '已完成') : job.status }}
                </t-tag></td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <details class="panel advanced-panel">
        <summary>高级用量明细（模型调用与原始计量）</summary>
        <p class="advanced-hint">这些维度用于诊断和成本分析，不是五个独立钱包。</p>
        <div class="quota-list">
          <div v-for="feature in overview.features" :key="feature.key" class="quota-row">
            <div class="quota-head">
              <div><strong>{{ featureLabel(feature.key) }}</strong><span>参考上限</span></div>
              <span>{{ formatQuantity(feature.used) }} / {{ formatQuantity(feature.limit) }} {{ unitLabel(feature.unit) }}</span>
            </div>
            <t-progress :percentage="usagePercent(feature)" :color="usagePercent(feature) >= 90 ? '#e34d59' : '#5b5bd6'" />
          </div>
        </div>
        <div v-if="usage.length" class="table-wrap raw-table">
          <table>
            <thead><tr><th>时间</th><th>能力</th><th>模型</th><th>原始用量</th><th>模式</th><th>扣费</th></tr></thead>
            <tbody><tr v-for="row in usage" :key="row.id">
              <td>{{ formatDate(row.created_at, true) }}</td><td>{{ featureLabel(row.feature_key) }}</td>
              <td>{{ row.model || row.provider || '—' }}</td>
              <td>{{ formatQuantity(row.quantity || 0) }} {{ unitLabel(row.unit) }}</td>
              <td>{{ modeLabel(row.billing_mode) }}</td><td>${{ formatMoney(row.cost_usd) }}</td>
            </tr></tbody>
          </table>
        </div>
      </details>

      <section v-if="isOwner" class="panel">
        <div class="panel-title">
          <div>
            <h3>账单</h3>
            <p>预付余额模式下，当前周期费用先从 Credit 扣除。</p>
          </div>
        </div>
        <div v-if="invoices.length === 0" class="empty-state">暂无已生成账单</div>
        <div v-else class="invoice-list">
          <div v-for="invoice in invoices" :key="invoice.id" class="invoice-row">
            <div><strong>{{ invoice.number || invoice.id }}</strong><span>{{ invoice.status }}</span></div>
            <div class="invoice-amount">{{ invoiceTotal(invoice) }}</div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useAuthStore } from '@/stores/auth'
import {
  cancelBillingPlan,
  createSandboxTopUp,
  getBillingInvoices,
  getBillingOverview,
  getBillingUsage,
  getBillingUsageJobs,
  undoBillingCancellation,
  upgradeBillingPlan,
  type BillingFeatureKey,
  type BillingFeatureOverview,
  type BillingOverview,
  type BillingUsageRow,
  type BillingUsageJob,
} from '@/api/billing'

const authStore = useAuthStore()
const overview = ref<BillingOverview | null>(null)
const usage = ref<BillingUsageRow[]>([])
const jobs = ref<BillingUsageJob[]>([])
const invoices = ref<any[]>([])
const loading = ref(false)
const actionLoading = ref('')
const errorMessage = ref('')
const partialWarning = ref('')
const topUpAmounts = [1, 5, 10] as const
const isOwner = computed(() => authStore.hasRole('owner') || authStore.canAccessAllTenants)
const statusText = computed(() => {
  if (!overview.value) return ''
  if (overview.value.cancel_scheduled) return '已安排在周期末取消'
  if (overview.value.status === 'active') return '正常使用中'
  if (overview.value.status === 'pending') return '正在开通'
  return overview.value.status
})

async function loadAll() {
  loading.value = true
  errorMessage.value = ''
  partialWarning.value = ''
  try {
    const overviewResponse = await getBillingOverview()
    overview.value = overviewResponse.data
  } catch (error: any) {
    errorMessage.value = error?.message || '读取套餐与用量失败'
    loading.value = false
    return
  }

  const [usageResult, jobsResult, invoiceResult] = await Promise.allSettled([
    getBillingUsage(),
    getBillingUsageJobs(),
    isOwner.value ? getBillingInvoices() : Promise.resolve(null),
  ])
  const unavailable: string[] = []
  if (usageResult.status === 'fulfilled') usage.value = usageResult.value.data || []
  else { usage.value = []; unavailable.push('模型调用明细') }
  if (jobsResult.status === 'fulfilled') jobs.value = jobsResult.value.data || []
  else { jobs.value = []; unavailable.push('AI 作业记录') }
  if (!isOwner.value) invoices.value = []
  else if (invoiceResult.status === 'fulfilled' && invoiceResult.value) {
    invoices.value = invoiceResult.value.data?.data || []
  } else {
    invoices.value = []
    unavailable.push('账单')
  }
  if (unavailable.length) {
    partialWarning.value = `${unavailable.join('、')}暂时不可用，套餐与余额仍可正常查看。`
  }
  loading.value = false
}

async function runAction(key: string, action: () => Promise<any>, success: string) {
  actionLoading.value = key
  try {
    const response = await action()
    if (response?.data) overview.value = response.data
    await loadAll()
    MessagePlugin.success(success)
  } catch (error: any) {
    MessagePlugin.error(error?.message || '操作失败')
  } finally {
    actionLoading.value = ''
  }
}

const topUp = (amount: 1 | 5 | 10) =>
  runAction(`topup-${amount}`, () => createSandboxTopUp(amount, crypto.randomUUID()), `已模拟充值 $${amount}`)
const upgrade = () => runAction('upgrade', upgradeBillingPlan, '已升级为 Pro')
const undoCancel = () => runAction('undo', undoBillingCancellation, '已撤销取消')
const cancelPlan = () => {
  if (!window.confirm('将在当前周期结束后取消 Pro，预付余额会保留。继续吗？')) return
  void runAction('cancel', cancelBillingPlan, '已安排在周期末取消')
}

function usagePercent(feature: BillingFeatureOverview) {
  if (!feature.limit) return 0
  return Math.min(100, Math.round((feature.used / feature.limit) * 100))
}
function featureLabel(key: BillingFeatureKey | string) {
  return ({
    lorelattice_llm_tokens: 'LLM / VLM',
    lorelattice_embedding_tokens: 'Embedding',
    lorelattice_rerank_tokens: 'Rerank',
    lorelattice_asr_seconds: 'ASR',
  } as Record<string, string>)[key] || key
}
function modeLabel(mode: string) {
  return ({ platform: '平台托管', byok: 'BYOK · 不扣费', local: '本地 · 不扣费', included: '套餐内含', mixed: '混合' } as Record<string, string>)[mode] || mode
}
function categoryLabel(category: string) {
  return ({ chat_agent: '对话 / Agent', document_indexing: '文档索引', image_processing: '图片处理', audio_transcription: '音频转写' } as Record<string, string>)[category] || category
}
function unitLabel(unit: string) { return unit === 'second' ? '秒' : 'Token' }
function formatQuantity(value?: number) { return new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 2 }).format(value || 0) }
function formatMoney(value?: number) { return Number(value || 0).toFixed(4).replace(/0+$/, '').replace(/\.$/, '') || '0' }
function formatDate(value?: string, withTime = false) {
  if (!value) return '—'
  return new Intl.DateTimeFormat('zh-CN', withTime
    ? { dateStyle: 'medium', timeStyle: 'short' }
    : { dateStyle: 'medium' }).format(new Date(value))
}
function invoiceTotal(invoice: any) {
  const amount = invoice?.totals?.total ?? invoice?.total ?? invoice?.amount ?? 0
  return `$${amount}`
}

watch(() => authStore.currentTenantId, () => void loadAll())
onMounted(loadAll)
</script>

<style scoped lang="less">
.billing-page { display: flex; flex-direction: column; gap: 18px; color: var(--td-text-color-primary); }
.page-header, .panel-title, .action-row, .quota-head, .invoice-row { display: flex; justify-content: space-between; gap: 16px; }
h2, h3, p { margin: 0; }
.page-header h2 { font-size: 22px; margin-bottom: 6px; }
.page-header p, .panel-title p, .summary-sub, .quota-foot, .action-label { color: var(--td-text-color-secondary); font-size: 13px; }
.summary-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.summary-card, .panel { border: 1px solid var(--td-component-border); border-radius: 10px; background: var(--td-bg-color-container); }
.summary-card { padding: 16px; display: flex; flex-direction: column; gap: 6px; }
.summary-card strong { font-size: 20px; }
.summary-label { color: var(--td-text-color-secondary); font-size: 13px; }
.panel { padding: 18px; }
.panel-title { margin-bottom: 16px; }
.panel-title h3 { font-size: 17px; margin-bottom: 4px; }
.quota-list { display: grid; gap: 18px; }
.credit-overview { display: flex; align-items: center; gap: 22px; flex-wrap: wrap; }
.credit-overview > div:first-child { display: flex; flex-direction: column; gap: 4px; min-width: 150px; }
.credit-overview > div:first-child span, .mode-stat span, .advanced-hint { color: var(--td-text-color-secondary); font-size: 13px; }
.credit-overview > div:first-child strong { font-size: 24px; }
.mode-stat { display: flex; align-items: center; gap: 8px; }
.advanced-panel summary { cursor: pointer; font-weight: 600; }
.advanced-hint { margin: 10px 0 18px; }
.raw-table { margin-top: 22px; }
.quota-head { align-items: center; margin-bottom: 7px; font-size: 13px; }
.quota-head div { display: flex; gap: 8px; align-items: baseline; }
.quota-head div span { color: var(--td-text-color-placeholder); }
.quota-foot { margin-top: 5px; text-align: right; }
.action-row { align-items: flex-end; }
.action-block { display: flex; flex-direction: column; gap: 8px; }
.button-group { display: flex; gap: 8px; flex-wrap: wrap; }
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th, td { padding: 10px 8px; text-align: left; border-bottom: 1px solid var(--td-component-stroke); white-space: nowrap; }
th { color: var(--td-text-color-secondary); font-weight: 500; }
.empty-state, .loading-state { padding: 28px; text-align: center; color: var(--td-text-color-secondary); }
.invoice-list { display: grid; }
.invoice-row { padding: 12px 0; border-bottom: 1px solid var(--td-component-stroke); }
.invoice-row > div:first-child { display: flex; flex-direction: column; gap: 3px; }
.invoice-row span { color: var(--td-text-color-secondary); font-size: 12px; }
.invoice-amount { font-weight: 600; }
@media (max-width: 800px) {
  .summary-grid { grid-template-columns: 1fr; }
  .action-row { flex-direction: column; align-items: stretch; }
}
</style>
