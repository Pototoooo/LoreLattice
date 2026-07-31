<template>
  <div class="billing-page">
    <div class="page-header">
      <div>
        <h2>套餐与用量</h2>
        <p>查看当前 workspace 的 AI 用量与套餐状态，无需进入 MeterForge Console。</p>
      </div>
      <t-button variant="outline" :loading="loading" @click="loadAll">刷新</t-button>
    </div>

    <t-alert v-if="errorMessage" theme="error" :message="errorMessage" />
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
          <span class="summary-label">预付余额</span>
          <strong>${{ formatMoney(overview.credit_balance_usd) }}</strong>
          <span class="summary-sub">用量必须同时满足额度和余额</span>
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
            <h3>本周期用量</h3>
            <p>每类 AI 能力独立计算额度，共享 USD 预付余额。</p>
          </div>
        </div>
        <div class="quota-list">
          <div v-for="feature in overview.features" :key="feature.key" class="quota-row">
            <div class="quota-head">
              <div>
                <strong>{{ featureLabel(feature.key) }}</strong>
                <span>${{ feature.unit_price_usd }}/{{ feature.unit }}</span>
              </div>
              <span>{{ formatQuantity(feature.used) }} / {{ formatQuantity(feature.limit) }} {{ unitLabel(feature.unit) }}</span>
            </div>
            <t-progress :percentage="usagePercent(feature)" :color="usagePercent(feature) >= 90 ? '#e34d59' : '#5b5bd6'" />
            <div class="quota-foot">剩余 {{ formatQuantity(feature.remaining) }} {{ unitLabel(feature.unit) }}</div>
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
            <h3>近期用量</h3>
            <p>记录实际 Provider 调用；“估算”表示 Provider 未返回 usage。</p>
          </div>
        </div>
        <div v-if="usage.length === 0" class="empty-state">暂无 AI 用量记录</div>
        <div v-else class="table-wrap">
          <table>
            <thead><tr><th>时间</th><th>能力</th><th>模型</th><th>用量</th><th>金额</th><th>状态</th></tr></thead>
            <tbody>
              <tr v-for="row in usage" :key="row.id">
                <td>{{ formatDate(row.created_at, true) }}</td>
                <td>{{ featureLabel(row.feature_key) }}</td>
                <td>{{ row.model || row.provider || '—' }}</td>
                <td>{{ formatQuantity(row.quantity || 0) }} {{ unitLabel(row.unit) }}</td>
                <td>${{ formatMoney(row.cost_usd) }}</td>
                <td><t-tag size="small" :theme="row.status === 'completed' ? 'success' : 'default'">
                  {{ row.status === 'completed' ? (row.estimated ? '已完成 · 估算' : '已完成') : row.status }}
                </t-tag></td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

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
  undoBillingCancellation,
  upgradeBillingPlan,
  type BillingFeatureKey,
  type BillingFeatureOverview,
  type BillingOverview,
  type BillingUsageRow,
} from '@/api/billing'

const authStore = useAuthStore()
const overview = ref<BillingOverview | null>(null)
const usage = ref<BillingUsageRow[]>([])
const invoices = ref<any[]>([])
const loading = ref(false)
const actionLoading = ref('')
const errorMessage = ref('')
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
  try {
    const [overviewResponse, usageResponse] = await Promise.all([
      getBillingOverview(),
      getBillingUsage(),
    ])
    overview.value = overviewResponse.data
    usage.value = usageResponse.data || []
    if (isOwner.value) {
      const invoiceResponse = await getBillingInvoices()
      invoices.value = invoiceResponse.data?.data || []
    } else {
      invoices.value = []
    }
  } catch (error: any) {
    errorMessage.value = error?.message || '读取套餐与用量失败'
  } finally {
    loading.value = false
  }
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
