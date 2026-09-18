<template>
  <div class="billing-page">
    <div class="page-header">
      <div><h2>余额与用量</h2><p>平台模型按量扣费，自备模型不扣余额。整个工作空间共享使用。</p></div>
      <t-button variant="outline" :loading="loading" @click="loadAll">刷新</t-button>
    </div>
    <t-alert v-if="errorMessage" theme="error" :message="errorMessage" />
    <t-alert v-if="overview && !overview.enabled" theme="info" message="此部署未启用计费。" />
    <template v-if="overview?.enabled">
      <section class="wallet panel">
        <div v-if="isOwner"><span>可用余额</span><strong class="balance">{{ money(overview.ai_credits.remaining_usd) }}</strong><p>平台模型调用前会预留费用，结束后按用量结算。</p></div>
        <div v-else><h3>由工作空间 Owner 管理余额</h3><p>你可以查看消费记录，充值请联系 Owner。</p></div>
        <div v-if="isOwner" class="top-up">
          <h3>支付宝充值</h3>
          <p v-if="!overview.payment_enabled">充值暂未开放，请联系管理员。</p>
          <div v-else class="buttons">
            <t-button v-for="amount in overview.top_up_amounts_fen" :key="amount" :disabled="!!actionLoading" :loading="actionLoading === String(amount)" @click="topUp(amount)">充值 ¥{{ amount / 100 }}</t-button>
          </div>
          <p v-if="overview.payment_enabled">无需订阅，付款确认后到账。</p>
        </div>
      </section>
      <section v-if="isOwner && activeOrder" class="panel payment-box">
        <h3>充值 ¥{{ (activeOrder.amount_fen / 100).toFixed(2) }}</h3>
        <p>{{ paymentStatus(activeOrder) }} · 订单 {{ activeOrder.id }}</p>
        <a v-if="checkoutLink" class="pay-link" :href="checkoutLink" target="_blank" rel="noopener noreferrer">前往支付宝付款</a>
        <p v-if="activeOrder.status === 'pending'">支付后本页会自动检查到账结果，也可以点击刷新。仅打开支付页面不会增加余额。</p>
      </section>
      <section class="panel">
        <h3>消费记录</h3><p>按一次问答、文档处理或转写汇总；可展开查看模型调用明细。</p>
        <div v-if="!jobs.length" class="empty">暂无消费记录</div>
        <div v-else class="table-wrap"><table>
          <thead><tr><th>时间</th><th>业务动作</th><th>计费方式</th><th>消费</th><th>状态</th></tr></thead>
          <tbody><tr v-for="job in jobs" :key="job.id"><td>{{ date(job.started_at) }}</td><td>{{ category(job.business_category) }}</td><td>{{ mode(job.billing_mode) }}</td><td>{{ money(job.cost_usd) }}</td><td>{{ usageStatus(job.status, job.estimated) }}</td></tr></tbody>
        </table></div>
      </section>
      <details class="panel"><summary>模型调用明细</summary>
        <div v-if="!usage.length" class="empty">暂无调用记录</div>
        <div v-else class="table-wrap"><table>
          <thead><tr><th>模型</th><th>用量</th><th>计费方式</th><th>消费</th><th>状态</th></tr></thead>
          <tbody><tr v-for="row in usage" :key="row.id"><td>{{ row.model || row.provider }}</td><td>{{ row.quantity ?? '—' }} {{ row.unit === 'second' ? '秒' : 'Token' }}</td><td>{{ mode(row.billing_mode) }}</td><td>{{ money(row.cost_usd) }}</td><td>{{ usageStatus(row.status, row.estimated) }}</td></tr></tbody>
        </table></div>
      </details>
      <section v-if="isOwner" class="panel">
        <h3>充值记录</h3><div v-if="!orders.length" class="empty">暂无充值订单</div>
        <div v-else class="table-wrap"><table>
          <thead><tr><th>时间</th><th>金额</th><th>状态</th><th></th></tr></thead>
          <tbody><tr v-for="order in orders" :key="order.id"><td>{{ date(order.created_at) }}</td><td>¥{{ (order.amount_fen / 100).toFixed(2) }}</td><td>{{ paymentStatus(order) }}</td><td><t-button v-if="order.status === 'pending'" variant="text" @click="inspectOrder(order.id)">查看订单</t-button></td></tr></tbody>
        </table></div>
      </section>
    </template>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { getBillingOverview, getBillingUsage, getBillingUsageJobs, createPayment, getPaymentOrders, getPaymentOrder, type BillingOverview, type BillingUsageRow, type BillingUsageJob, type PaymentOrder } from '@/api/billing'

const authStore = useAuthStore()
const isOwner = computed(() => authStore.hasRole('owner') || authStore.isSystemAdmin)
const overview = ref<BillingOverview | null>(null)
const usage = ref<BillingUsageRow[]>([])
const jobs = ref<BillingUsageJob[]>([])
const orders = ref<PaymentOrder[]>([])
const activeOrder = ref<PaymentOrder | null>(null)
const loading = ref(false)
const actionLoading = ref('')
const errorMessage = ref('')
let generation = 0
let disposed = false
let polling = false
let timer: ReturnType<typeof setInterval> | undefined
// Retain the same key if a create response is lost. A second click cannot double-create.
const pendingKeys = new Map<number, string>()
const checkoutLink = computed(() => {
  if (!activeOrder.value?.checkout_url || activeOrder.value.status !== 'pending') return ''
  try {
    const url = new URL(activeOrder.value.checkout_url)
    return url.origin === 'https://openapi.alipay.com' && url.pathname === '/gateway.do' ? url.href : ''
  } catch { return '' }
})
async function loadAll() {
  const current = ++generation
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await getBillingOverview()
    if (disposed || current !== generation) return
    overview.value = result.data
    if (!result.data.enabled) return
    const results = await Promise.allSettled([getBillingUsage(), getBillingUsageJobs(), isOwner.value ? getPaymentOrders() : Promise.resolve(null)])
    if (disposed || current !== generation) return
    const [calls, jobRows, paymentRows] = results
    usage.value = calls.status === 'fulfilled' ? calls.value.data || [] : []
    jobs.value = jobRows.status === 'fulfilled' ? jobRows.value.data || [] : []
    orders.value = paymentRows.status === 'fulfilled' && paymentRows.value ? paymentRows.value.data || [] : []
    if (results.some(r => r.status === 'rejected')) errorMessage.value = '部分记录读取失败，请刷新重试。'
  } catch (error: any) {
    if (!disposed && current === generation) errorMessage.value = error?.message || '读取余额失败'
  } finally { if (current === generation) loading.value = false }
}
async function topUp(amount: number) {
  const tenant = authStore.currentTenantId
  actionLoading.value = String(amount)
  errorMessage.value = ''
  if (!pendingKeys.has(amount)) pendingKeys.set(amount, crypto.randomUUID())
  try {
    const result = await createPayment(amount, pendingKeys.get(amount)!)
    if (disposed || tenant !== authStore.currentTenantId) return
    activeOrder.value = result.data
    pendingKeys.delete(amount)
    await loadAll()
  } catch (error: any) { if (tenant === authStore.currentTenantId) errorMessage.value = error?.message || '创建充值订单失败，请重试' }
  finally { if (tenant === authStore.currentTenantId) actionLoading.value = '' }
}
async function inspectOrder(id: string) {
  const tenant = authStore.currentTenantId
  try {
    const result = await getPaymentOrder(id)
    if (disposed || tenant !== authStore.currentTenantId) return
    activeOrder.value = result.data
    if (result.data.status !== 'pending') await loadAll()
  } catch (error: any) { if (tenant === authStore.currentTenantId) errorMessage.value = error?.message || '检查付款结果失败，请刷新重试' }
}
async function poll() {
  if (polling || loading.value || document.hidden || !isOwner.value || !overview.value?.enabled) return
  const id = activeOrder.value?.status === 'pending' ? activeOrder.value.id : orders.value.find(o => o.status === 'pending' && Date.parse(o.expires_at) > Date.now())?.id
  if (!id) return
  polling = true
  try { await inspectOrder(id) } finally { polling = false }
}
function money(usd?: number) {
  if (usd === undefined) return '—'
  const cny = usd * (overview.value?.cny_per_usd || 0)
  return `¥${cny.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 4 })}`
}
function date(raw: string) { return new Date(raw).toLocaleString('zh-CN') }
function mode(value: string) { return value === 'platform' ? '平台模型' : value === 'mixed' ? '平台与自备模型' : '自备模型' }
function category(value: string) { return ({ chat_agent: '问答 / Agent', document_indexing: '文档处理', image_processing: '图片处理', audio_transcription: '音频转写' } as Record<string, string>)[value] || value }
function usageStatus(status: string, estimated: boolean) {
  return ({ completed: estimated ? '已结算（估算）' : '已结算', recovered: '暂结待核查', reserved: '处理中', released: '未扣费', partial: '部分完成' } as Record<string, string>)[status] || status
}
function paymentStatus(order: PaymentOrder) { return order.status === 'paid' ? '已到账' : order.status === 'closed' ? '已关闭' : Date.parse(order.expires_at) < Date.now() ? '支付窗口已结束，待核实' : '待付款' }
watch(() => authStore.currentTenantId, () => {
  generation++; overview.value = null; activeOrder.value = null; orders.value = []; jobs.value = []; usage.value = []; pendingKeys.clear(); actionLoading.value = ''
  void loadAll()
})
watch(isOwner, owner => { if (!owner) { activeOrder.value = null; orders.value = []; pendingKeys.clear() } })
onMounted(() => { void loadAll(); timer = setInterval(() => void poll(), 5000) })
onUnmounted(() => { disposed = true; generation++; clearInterval(timer) })
</script>
<style scoped lang="less">
.billing-page { display: flex; flex-direction: column; gap: 20px; color: var(--td-text-color-primary); }
.page-header, .wallet { display: flex; justify-content: space-between; gap: 24px; align-items: flex-start; }
h2, h3, p { margin: 0; }
h2 { font-size: 22px; } h3 { font-size: 16px; margin-bottom: 10px; }
p { font-size: 13px; color: var(--td-text-color-secondary); line-height: 1.7; margin-top: 8px; }
.panel { padding: 24px; border: 1px solid var(--td-component-border); border-radius: 12px; background: var(--td-bg-color-container); }
.balance { display: block; font-size: 36px; font-weight: 600; margin-top: 10px; font-variant-numeric: tabular-nums; }
.buttons { display: flex; gap: 8px; flex-wrap: wrap; }
.table-wrap { overflow-x: auto; margin-top: 14px; }
table { width: 100%; border-collapse: collapse; text-align: left; font-size: 13px; }
th, td { padding: 12px 8px; border-bottom: 1px solid var(--td-component-border); white-space: nowrap; }
th { color: var(--td-text-color-secondary); font-weight: 500; }
.empty { padding: 24px 0; color: var(--td-text-color-placeholder); }
summary { cursor: pointer; font-weight: 500; }
.pay-link { display: inline-block; padding: 10px 18px; background: var(--td-brand-color); color: white; border-radius: 6px; text-decoration: none; margin-top: 14px; }
@media (max-width: 760px) { .wallet { flex-direction: column; } .panel { padding: 16px; } }
</style>
