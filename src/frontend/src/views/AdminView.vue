<!-- 文件作用：管理审核控制台（S-14 治理闭环的前端一环）。举报列表按状态筛选，
     支持"下架内容并办结 / 仅记录办结 / 驳回举报"，附审计链查询；非管理员访问
     展示明确的权限提示空态，不暴露任何数据。所有数据来自 /admin 接口，不伪造。 -->
<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { adminApi, type ReportStatusFilter } from '@/api/adminApi'
import PageError from '@/components/feedback/PageError.vue'
import PageLoading from '@/components/feedback/PageLoading.vue'
import type { AuditEvent, Report } from '@/types'

const STATUS_TABS: Array<{ key: ReportStatusFilter; label: string }> = [
  { key: 'pending', label: '待处理' },
  { key: 'reviewed', label: '已办结' },
  { key: 'rejected', label: '已驳回' }
]

const REASON_LABELS: Record<string, string> = {
  spam: '垃圾内容',
  abuse: '辱骂攻击',
  nsfw: '不当内容',
  other: '其他'
}

const activeTab = ref<ReportStatusFilter>('pending')
const reports = ref<Report[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)
const forbidden = ref(false)
const loadFailed = ref(false)
const busyReportId = ref<number | null>(null)
const noteDrafts = reactive<Record<number, string>>({})

const auditEvents = ref<AuditEvent[]>([])
const auditTotal = ref(0)
const auditPage = ref(1)
const auditLoading = ref(false)
const auditOpen = ref(false)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const auditTotalPages = computed(() => Math.max(1, Math.ceil(auditTotal.value / 10)))

const targetLabel = (report: Report) => (report.targetType === 'prompt' ? '提示词' : report.targetType === 'comment' ? '评论' : report.targetType)
const reasonLabel = (report: Report) => REASON_LABELS[report.reason] ?? report.reason

const parseMeta = (meta: string): Record<string, unknown> => {
  try {
    return JSON.parse(meta) as Record<string, unknown>
  } catch {
    return {}
  }
}

const metaSummary = (event: AuditEvent): string => {
  const meta = parseMeta(event.metadata)
  const parts: string[] = []
  if (typeof meta.status === 'string' && meta.status) {
    parts.push(`状态 ${meta.status}`)
  }
  if (typeof meta.action === 'string' && meta.action && meta.action !== 'none') {
    parts.push(meta.action === 'remove' ? '已下架内容' : `动作 ${meta.action}`)
  }
  if (typeof meta.note === 'string' && meta.note) {
    parts.push(`备注：${meta.note}`)
  }
  return parts.join(' · ')
}

const loadReports = async (tab = activeTab.value, targetPage = page.value) => {
  loading.value = true
  loadFailed.value = false
  try {
    const response = await adminApi.listReports(tab, targetPage, pageSize)
    reports.value = response.data.list
    total.value = response.data.total
    page.value = response.data.page
    forbidden.value = false
  } catch (error) {
    const status = (error as { response?: { status?: number } }).response?.status
    if (status === 403) {
      forbidden.value = true
      reports.value = []
    } else {
      loadFailed.value = true
    }
  } finally {
    loading.value = false
  }
}

const switchTab = (tab: ReportStatusFilter) => {
  activeTab.value = tab
  page.value = 1
  void loadReports(tab, 1)
}

const review = async (report: Report, status: 'reviewed' | 'rejected', action: 'none' | 'remove') => {
  if (busyReportId.value !== null) {
    return
  }
  busyReportId.value = report.id
  try {
    await adminApi.reviewReport(report.id, {
      status,
      action,
      note: (noteDrafts[report.id] ?? '').trim()
    })
    await loadReports()
  } finally {
    busyReportId.value = null
  }
}

const loadAudit = async (targetPage = auditPage.value) => {
  auditLoading.value = true
  try {
    const response = await adminApi.listAuditEvents(targetPage, 10)
    auditEvents.value = response.data.list
    auditTotal.value = response.data.total
    auditPage.value = response.data.page
  } finally {
    auditLoading.value = false
  }
}

const toggleAudit = async () => {
  auditOpen.value = !auditOpen.value
  if (auditOpen.value && auditEvents.value.length === 0) {
    await loadAudit(1)
  }
}

onMounted(() => {
  void loadReports()
})
</script>

<template>
  <div class="admin-page">
    <div class="admin-container">
      <header class="admin-head">
        <div>
          <p class="section-eyebrow">
            治理后台
          </p>
          <h1 class="admin-head__title">
            审核控制台
          </h1>
        </div>
        <button
          type="button"
          class="admin-audit-toggle"
          :aria-expanded="auditOpen"
          @click="toggleAudit"
        >
          {{ auditOpen ? '收起审计链' : '查看审计链' }}
        </button>
      </header>

      <PageLoading
        v-if="loading && reports.length === 0"
        label="正在加载举报"
        variant="blocks"
      />

      <PageError
        v-else-if="forbidden"
        kind="empty"
        title="需要管理员角色"
        description="当前账号没有审核权限。如需开通，请联系站点管理员在服务器侧配置 user_roles。"
        action-label="返回首页"
        @action="$router.push('/')"
      />

      <PageError
        v-else-if="loadFailed && reports.length === 0"
        kind="error"
        title="举报列表加载失败"
        description="暂时无法加载举报列表，请稍后重试。"
        action-label="重新加载"
        :busy="loading"
        @action="loadReports()"
      />

      <template v-else>
        <section
          class="admin-reports"
          aria-labelledby="admin-reports-title"
        >
          <div class="admin-section-head">
            <h2
              id="admin-reports-title"
              class="admin-section-title"
            >
              举报处理
            </h2>
            <div
              class="admin-tabs"
              role="tablist"
              aria-label="举报状态筛选"
            >
              <button
                v-for="tab in STATUS_TABS"
                :key="tab.key"
                type="button"
                role="tab"
                class="admin-tab"
                :class="{ 'admin-tab--active': activeTab === tab.key }"
                :aria-selected="activeTab === tab.key"
                @click="switchTab(tab.key)"
              >
                {{ tab.label }}
              </button>
            </div>
          </div>

          <ul
            v-if="reports.length"
            class="admin-report-list"
          >
            <li
              v-for="report in reports"
              :key="report.id"
              class="admin-report"
            >
              <div class="admin-report__meta">
                <span class="admin-report__target">#{{ report.id }} · {{ targetLabel(report) }} {{ report.targetId }}</span>
                <span class="admin-report__reason">{{ reasonLabel(report) }}</span>
                <time class="admin-report__time">{{ report.createdAt }}</time>
              </div>
              <p
                v-if="report.detail"
                class="admin-report__detail"
              >
                {{ report.detail }}
              </p>
              <p
                v-if="report.status !== 'pending'"
                class="admin-report__reviewed"
              >
                处理人 #{{ report.reviewedBy }} · {{ report.reviewNote || '（无备注）' }}
              </p>

              <div
                v-if="report.status === 'pending'"
                class="admin-report__actions"
              >
                <input
                  v-model="noteDrafts[report.id]"
                  type="text"
                  class="admin-note-input"
                  placeholder="处理备注（可选）"
                  aria-label="处理备注"
                  maxlength="200"
                >
                <button
                  type="button"
                  class="admin-btn admin-btn--primary"
                  :disabled="busyReportId === report.id"
                  @click="review(report, 'reviewed', report.targetType === 'prompt' ? 'remove' : 'none')"
                >
                  {{ busyReportId === report.id ? '处理中…' : report.targetType === 'prompt' ? '下架内容并办结' : '标记已处理' }}
                </button>
                <button
                  type="button"
                  class="admin-btn"
                  :disabled="busyReportId === report.id"
                  @click="review(report, 'rejected', 'none')"
                >
                  驳回举报
                </button>
              </div>
            </li>
          </ul>

          <p
            v-else
            class="admin-empty"
          >
            当前状态没有举报。
          </p>

          <div
            v-if="totalPages > 1"
            class="admin-pager"
          >
            <button
              type="button"
              class="admin-btn"
              :disabled="page <= 1 || loading"
              @click="loadReports(activeTab, page - 1)"
            >
              上一页
            </button>
            <span class="admin-pager__info">{{ page }} / {{ totalPages }}</span>
            <button
              type="button"
              class="admin-btn"
              :disabled="page >= totalPages || loading"
              @click="loadReports(activeTab, page + 1)"
            >
              下一页
            </button>
          </div>
        </section>

        <section
          v-if="auditOpen"
          class="admin-audit"
          aria-labelledby="admin-audit-title"
        >
          <h2
            id="admin-audit-title"
            class="admin-section-title"
          >
            审计链
          </h2>
          <PageLoading
            v-if="auditLoading"
            label="正在加载审计事件"
            variant="blocks"
          />
          <ul
            v-else-if="auditEvents.length"
            class="admin-audit-list"
          >
            <li
              v-for="event in auditEvents"
              :key="event.id"
              class="admin-audit-event"
            >
              <div class="admin-audit-event__head">
                <span class="admin-audit-event__id">#{{ event.id }}</span>
                <span class="admin-audit-event__action">{{ event.action }}</span>
                <span class="admin-audit-event__target">{{ event.targetType }} {{ event.targetId }}</span>
                <time class="admin-audit-event__time">{{ event.createdAt }}</time>
              </div>
              <div class="admin-audit-event__meta">
                操作人 #{{ event.actorId }} ·
                <span title="本事件哈希">hash {{ event.eventHash.slice(0, 12) }}…</span>
                ←
                <span title="上一事件哈希">prev {{ event.prevHash ? event.prevHash.slice(0, 12) + '…' : '（起点）' }}</span>
              </div>
              <p
                v-if="metaSummary(event)"
                class="admin-audit-event__note"
              >
                {{ metaSummary(event) }}
              </p>
            </li>
          </ul>
          <p
            v-else
            class="admin-empty"
          >
            还没有审计事件。
          </p>
          <div
            v-if="auditTotalPages > 1"
            class="admin-pager"
          >
            <button
              type="button"
              class="admin-btn"
              :disabled="auditPage <= 1 || auditLoading"
              @click="loadAudit(auditPage - 1)"
            >
              上一页
            </button>
            <span class="admin-pager__info">{{ auditPage }} / {{ auditTotalPages }}</span>
            <button
              type="button"
              class="admin-btn"
              :disabled="auditPage >= auditTotalPages || auditLoading"
              @click="loadAudit(auditPage + 1)"
            >
              下一页
            </button>
          </div>
        </section>
      </template>
    </div>
  </div>
</template>

<style scoped>
.admin-page {
  @apply view-page;
}

.admin-container {
  @apply view-container;
}

.admin-head {
  @apply flex items-end justify-between gap-4 py-10;
}

.admin-head__title {
  @apply mt-1 text-2xl font-semibold text-[var(--prompt-text)] sm:text-3xl;
}

.admin-audit-toggle {
  @apply inline-flex min-h-[40px] items-center rounded-full border bg-[var(--prompt-surface)] px-4 text-sm font-medium text-[var(--prompt-text-muted)] transition hover:border-[var(--prompt-border-strong)] hover:text-[var(--prompt-text)];
  border-color: var(--prompt-border);
}

.admin-section-head {
  @apply flex flex-wrap items-end justify-between gap-3;
}

.admin-section-title {
  @apply text-lg font-semibold text-[var(--prompt-text)] sm:text-xl;
}

.admin-tabs {
  @apply flex gap-2;
}

.admin-tab {
  @apply rounded-full border px-4 py-2 text-sm font-medium text-[var(--prompt-text-muted)] transition;
  border-color: var(--prompt-border);
}

.admin-tab--active {
  background-color: var(--prompt-primary);
  border-color: var(--prompt-primary);
  color: var(--prompt-primary-contrast);
  font-weight: 600;
}

.admin-report-list {
  @apply mt-5 grid gap-4;
}

.admin-report {
  @apply rounded-[var(--prompt-radius)] border bg-[var(--prompt-surface)] p-5;
  border-color: var(--prompt-border);
  box-shadow: var(--prompt-shadow-1);
}

.admin-report__meta {
  @apply flex flex-wrap items-center gap-x-3 gap-y-1 text-sm;
}

.admin-report__target {
  @apply font-semibold text-[var(--prompt-text)];
}

.admin-report__reason {
  @apply rounded-full bg-[var(--prompt-surface-muted)] px-2.5 py-0.5 text-xs text-[var(--prompt-text-muted)];
}

.admin-report__time {
  @apply text-xs text-[var(--prompt-text-faint)];
}

.admin-report__detail {
  @apply mt-2 text-sm leading-6 text-[var(--prompt-text-muted)];
}

.admin-report__reviewed {
  @apply mt-2 text-sm text-[var(--prompt-text-faint)];
}

.admin-report__actions {
  @apply mt-4 flex flex-wrap items-center gap-3;
}

.admin-note-input {
  @apply h-10 min-w-0 flex-1 rounded-full border bg-[var(--prompt-surface-muted)] px-4 text-sm text-[var(--prompt-text)] outline-none placeholder:text-[var(--prompt-text-faint)];
  border-color: var(--prompt-border);
}

.admin-btn {
  @apply inline-flex min-h-[40px] items-center rounded-full border bg-[var(--prompt-surface)] px-4 text-sm font-medium text-[var(--prompt-text-muted)] transition hover:border-[var(--prompt-border-strong)] hover:text-[var(--prompt-text)] disabled:cursor-wait disabled:opacity-60;
  border-color: var(--prompt-border);
}

.admin-btn--primary {
  background-color: var(--prompt-primary);
  border-color: var(--prompt-primary);
  color: var(--prompt-primary-contrast);
  font-weight: 600;
}

.admin-btn--primary:hover:not(:disabled) {
  background-color: var(--prompt-primary-hover);
  color: var(--prompt-primary-contrast);
}

.admin-empty {
  @apply rounded-[var(--prompt-radius-lg)] border border-dashed px-6 py-12 text-center text-sm text-[var(--prompt-text-muted)];
  border-color: var(--prompt-border);
}

.admin-pager {
  @apply mt-6 flex items-center justify-center gap-4;
}

.admin-pager__info {
  @apply text-sm tabular-nums text-[var(--prompt-text-faint)];
}

.admin-audit {
  @apply mt-12;
}

.admin-audit-list {
  @apply mt-4 grid gap-3;
}

.admin-audit-event {
  @apply rounded-[var(--prompt-radius-sm)] border bg-[var(--prompt-surface)] p-4;
  border-color: var(--prompt-border);
}

.admin-audit-event__head {
  @apply flex flex-wrap items-center gap-x-3 gap-y-1 text-sm;
}

.admin-audit-event__id {
  @apply font-semibold tabular-nums text-[var(--prompt-text)];
}

.admin-audit-event__action {
  @apply rounded-full bg-[var(--prompt-surface-muted)] px-2.5 py-0.5 text-xs text-[var(--prompt-text-muted)];
}

.admin-audit-event__target {
  @apply text-[var(--prompt-text-muted)];
}

.admin-audit-event__time {
  @apply text-xs text-[var(--prompt-text-faint)];
}

.admin-audit-event__meta {
  @apply mt-1.5 text-xs text-[var(--prompt-text-faint)];
}

.admin-audit-event__note {
  @apply mt-1.5 text-sm leading-6 text-[var(--prompt-text-muted)];
}

@media (prefers-reduced-motion: reduce) {
  .admin-tab,
  .admin-btn,
  .admin-audit-toggle {
    transition: none;
  }
}
</style>
