<!-- 文件作用：首次部署初始化向导。环境检查 → 创建管理员账号 → 站点信息 →
     邮件 SMTP 配置（可测试/可跳过）→ 确认提交，一次性完成后引导登录。
     后端 /setup/initialize 具有原子一次性语义，重复提交返回 409。 -->
<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { NButton, NCheckbox, NInput } from 'naive-ui'
import {
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
  CircleAlert,
  Database,
  Loader2,
  Mail,
  Server,
  ShieldCheck
} from '@lucide/vue'
import PageLoading from '@/components/feedback/PageLoading.vue'
import PageError from '@/components/feedback/PageError.vue'
import { setupApi, type SetupStatus } from '@/api/setupApi'

const router = useRouter()
const message = useMessage()

const loadingStatus = ref(true)
const loadError = ref('')
const status = ref<SetupStatus | null>(null)
const submitting = ref(false)
const testingSMTP = ref(false)
const smtpTested = ref(false)
const initialized = ref(false)

const step = ref(0)
const STEP_TITLES = ['环境检查', '管理员账号', '站点信息', '邮件配置', '确认初始化']

const form = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  siteName: 'PromptOS',
  siteDescription: '',
  registrationEnabled: true,
  smtpEnabled: true,
  smtp: {
    host: '',
    port: '587',
    user: '',
    password: '',
    from: ''
  },
  testRecipient: ''
})

const emailValid = computed(() => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email.trim()))
const passwordValid = computed(() => form.password.length >= 8 && form.password.length <= 72)
const adminStepValid = computed(
  () => form.username.trim().length >= 2 && emailValid.value && passwordValid.value && form.password === form.confirmPassword
)
const smtpFormValid = computed(() => {
  if (!form.smtpEnabled) {
    return true
  }
  return form.smtp.host.trim() !== '' && form.smtp.from.trim() !== ''
})

const loadStatus = async () => {
  loadingStatus.value = true
  loadError.value = ''
  try {
    const response = await setupApi.getStatus()
    status.value = response.data
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '无法连接后端服务'
  } finally {
    loadingStatus.value = false
  }
}

const goLogin = () => {
  void router.push('/login')
}

const next = () => {
  if (step.value === 1 && !adminStepValid.value) {
    message.warning('请完整填写管理员信息：用户名至少 2 位、邮箱合法、密码至少 8 位且两次一致')
    return
  }
  step.value = Math.min(step.value + 1, STEP_TITLES.length - 1)
}

const prev = () => {
  step.value = Math.max(step.value - 1, 0)
}

const sendTestEmail = async () => {
  if (!smtpFormValid.value) {
    message.warning('请先填写 SMTP 服务器地址和发件人邮箱')
    return
  }
  const recipient = form.testRecipient.trim() || form.email.trim()
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(recipient)) {
    message.warning('请填写有效的测试收件邮箱')
    return
  }
  testingSMTP.value = true
  try {
    await setupApi.testSMTP({ smtp: { ...form.smtp }, recipient })
    smtpTested.value = true
    message.success(`测试邮件已发送至 ${recipient}，请查收`)
  } catch {
    smtpTested.value = false
  } finally {
    testingSMTP.value = false
  }
}

const submit = async () => {
  submitting.value = true
  try {
    await setupApi.initialize({
      username: form.username.trim(),
      email: form.email.trim(),
      password: form.password,
      siteName: form.siteName.trim() || 'PromptOS',
      siteDescription: form.siteDescription.trim(),
      smtp:
        form.smtpEnabled && smtpFormValid.value
          ? { ...form.smtp, host: form.smtp.host.trim(), from: form.smtp.from.trim() }
          : undefined,
      registrationEnabled: form.registrationEnabled
    })
    initialized.value = true
    message.success('初始化完成，请使用管理员账号登录')
  } catch {
    // request 拦截器已弹出具体错误（如 409 SETUP_COMPLETED）
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  void loadStatus()
})
</script>

<template>
  <div class="setup-page">
    <div class="setup-card">
      <div class="setup-header">
        <span class="setup-brand">PromptOS</span>
        <h1 class="setup-title">
          系统初始化
        </h1>
        <p class="setup-subtitle">
          首次部署引导：创建管理员、配置站点与邮件服务
        </p>
      </div>

      <PageLoading v-if="loadingStatus" />

      <PageError
        v-else-if="loadError"
        kind="error"
        title="无法获取初始化状态"
        :description="loadError"
        action-label="重试"
        :on-action="loadStatus"
      />

      <!-- 已初始化：包括本页提交成功后的终态 -->
      <div
        v-else-if="initialized || status?.initialized"
        class="setup-done"
      >
        <CheckCircle2 class="setup-done__icon" />
        <h2 class="setup-done__title">
          {{ initialized ? '初始化完成' : '系统已完成初始化' }}
        </h2>
        <p class="setup-done__desc">
          管理员账号已创建。日常站点与邮件配置可在登录后进入「工作台 → 审核控制台 → 站点设置」调整。
        </p>
        <div class="setup-actions">
          <NButton
            type="primary"
            size="large"
            @click="goLogin"
          >
            前往登录
          </NButton>
          <RouterLink
            to="/"
            class="setup-link"
          >
            返回首页
          </RouterLink>
        </div>
      </div>

      <template v-else>
        <!-- 步骤条 -->
        <ol class="setup-steps">
          <li
            v-for="(title, index) in STEP_TITLES"
            :key="title"
            class="setup-step"
            :class="{
              'setup-step--active': index === step,
              'setup-step--done': index < step
            }"
          >
            <span class="setup-step__index">{{ index < step ? '✓' : index + 1 }}</span>
            <span class="setup-step__label">{{ title }}</span>
          </li>
        </ol>

        <!-- 步骤 0：环境检查 -->
        <section
          v-if="step === 0"
          class="setup-section"
        >
          <ul class="setup-checklist">
            <li class="setup-check">
              <Database class="setup-check__icon" />
              <div class="setup-check__body">
                <p class="setup-check__name">
                  存储模式
                </p>
                <p class="setup-check__value">
                  {{ status?.storageMode === 'mysql' ? 'MySQL（持久化就绪）' : '内存模式（仅开发环境）' }}
                </p>
              </div>
              <span
                class="setup-badge"
                :class="status?.storageMode === 'mysql' ? 'setup-badge--ok' : 'setup-badge--warn'"
              >
                {{ status?.storageMode === 'mysql' ? '正常' : '警告' }}
              </span>
            </li>
            <li class="setup-check">
              <Server class="setup-check__icon" />
              <div class="setup-check__body">
                <p class="setup-check__name">
                  运行环境
                </p>
                <p class="setup-check__value">
                  {{ status?.environment || 'unknown' }}
                </p>
              </div>
              <span class="setup-badge setup-badge--ok">{{ status?.environment }}</span>
            </li>
            <li class="setup-check">
              <Mail class="setup-check__icon" />
              <div class="setup-check__body">
                <p class="setup-check__name">
                  邮件服务（环境变量）
                </p>
                <p class="setup-check__value">
                  {{ status?.smtpConfigured ? '已通过 SMTP_* 环境变量配置' : '未配置，可在下一步设置' }}
                </p>
              </div>
              <span
                class="setup-badge"
                :class="status?.smtpConfigured ? 'setup-badge--ok' : 'setup-badge--muted'"
              >
                {{ status?.smtpConfigured ? '已配置' : '待配置' }}
              </span>
            </li>
            <li class="setup-check">
              <ShieldCheck class="setup-check__icon" />
              <div class="setup-check__body">
                <p class="setup-check__name">
                  注册通道
                </p>
                <p class="setup-check__value">
                  {{ status?.emailAuthEnabled ? '邮箱验证码注册' : '当前关闭邮箱验证（环境配置）' }}
                </p>
              </div>
              <span class="setup-badge setup-badge--muted">信息</span>
            </li>
          </ul>
          <CircleAlert
            v-if="status?.storageMode !== 'mysql'"
            class="setup-hint__icon"
          />
          <p
            v-if="status?.storageMode !== 'mysql'"
            class="setup-hint"
          >
            当前为内存存储：初始化数据不会持久化。生产部署请先确认 MySQL/Redis 已就绪。
          </p>
        </section>

        <!-- 步骤 1：管理员账号 -->
        <section
          v-else-if="step === 1"
          class="setup-section"
        >
          <div class="setup-form">
            <label class="setup-field">
              <span class="setup-field__label">管理员用户名</span>
              <NInput
                v-model:value="form.username"
                placeholder="例如 founder"
                :input-props="{ autocomplete: 'username' }"
              />
            </label>
            <label class="setup-field">
              <span class="setup-field__label">登录邮箱</span>
              <NInput
                v-model:value="form.email"
                placeholder="admin@example.com"
                :input-props="{ autocomplete: 'email', type: 'email' }"
              />
            </label>
            <label class="setup-field">
              <span class="setup-field__label">密码（至少 8 位）</span>
              <NInput
                v-model:value="form.password"
                type="password"
                show-password-on="click"
                placeholder="设置强密码"
                :input-props="{ autocomplete: 'new-password' }"
              />
            </label>
            <label class="setup-field">
              <span class="setup-field__label">确认密码</span>
              <NInput
                v-model:value="form.confirmPassword"
                type="password"
                show-password-on="click"
                placeholder="再次输入密码"
                :input-props="{ autocomplete: 'new-password' }"
              />
            </label>
          </div>
        </section>

        <!-- 步骤 2：站点信息 -->
        <section
          v-else-if="step === 2"
          class="setup-section"
        >
          <div class="setup-form">
            <label class="setup-field">
              <span class="setup-field__label">站点名称</span>
              <NInput
                v-model:value="form.siteName"
                placeholder="PromptOS"
                :maxlength="60"
              />
            </label>
            <label class="setup-field">
              <span class="setup-field__label">站点描述（可选）</span>
              <NInput
                v-model:value="form.siteDescription"
                type="textarea"
                placeholder="一句话介绍这个社区"
                :maxlength="200"
                :rows="2"
              />
            </label>
            <div class="setup-field setup-field--row">
              <NCheckbox v-model:checked="form.registrationEnabled">
                开放新用户注册
              </NCheckbox>
              <span class="setup-field__hint">关闭后仅管理员可邀请注册，可随时在站点设置中调整</span>
            </div>
          </div>
        </section>

        <!-- 步骤 3：邮件配置 -->
        <section
          v-else-if="step === 3"
          class="setup-section"
        >
          <div class="setup-form">
            <div class="setup-field setup-field--row">
              <NCheckbox v-model:checked="form.smtpEnabled">
                配置邮件服务（验证码/通知依赖）
              </NCheckbox>
              <span class="setup-field__hint">未配置时将沿用环境变量 SMTP_*；两者都缺省则验证码邮件不可用</span>
            </div>
            <template v-if="form.smtpEnabled">
              <div class="setup-grid-2">
                <label class="setup-field">
                  <span class="setup-field__label">SMTP 服务器</span>
                  <NInput
                    v-model:value="form.smtp.host"
                    placeholder="smtp.example.com"
                  />
                </label>
                <label class="setup-field">
                  <span class="setup-field__label">端口</span>
                  <NInput
                    v-model:value="form.smtp.port"
                    placeholder="587"
                  />
                </label>
              </div>
              <div class="setup-grid-2">
                <label class="setup-field">
                  <span class="setup-field__label">用户名（可选）</span>
                  <NInput
                    v-model:value="form.smtp.user"
                    placeholder="noreply@example.com"
                  />
                </label>
                <label class="setup-field">
                  <span class="setup-field__label">密码/授权码（可选）</span>
                  <NInput
                    v-model:value="form.smtp.password"
                    type="password"
                    show-password-on="click"
                    placeholder="SMTP 凭据"
                  />
                </label>
              </div>
              <label class="setup-field">
                <span class="setup-field__label">发件人邮箱</span>
                <NInput
                  v-model:value="form.smtp.from"
                  placeholder="noreply@example.com"
                />
              </label>
              <label class="setup-field">
                <span class="setup-field__label">测试收件邮箱（默认使用管理员邮箱）</span>
                <NInput
                  v-model:value="form.testRecipient"
                  :placeholder="form.email || 'admin@example.com'"
                />
              </label>
              <div class="setup-field--row setup-test">
                <NButton
                  :loading="testingSMTP"
                  :disabled="!smtpFormValid"
                  @click="sendTestEmail"
                >
                  发送测试邮件
                </NButton>
                <span
                  v-if="smtpTested"
                  class="setup-test__ok"
                >✓ 已发送</span>
              </div>
            </template>
          </div>
        </section>

        <!-- 步骤 4：确认 -->
        <section
          v-else
          class="setup-section"
        >
          <dl class="setup-summary">
            <div class="setup-summary__row">
              <dt>管理员</dt>
              <dd>{{ form.username }}（{{ form.email }}）</dd>
            </div>
            <div class="setup-summary__row">
              <dt>站点</dt>
              <dd>{{ form.siteName || 'PromptOS' }}</dd>
            </div>
            <div class="setup-summary__row">
              <dt>开放注册</dt>
              <dd>{{ form.registrationEnabled ? '开放' : '关闭' }}</dd>
            </div>
            <div class="setup-summary__row">
              <dt>邮件服务</dt>
              <dd>
                {{ form.smtpEnabled && smtpFormValid ? `${form.smtp.host}:${form.smtp.port}` : '沿用环境变量（未配置）' }}
              </dd>
            </div>
          </dl>
          <p class="setup-hint">
            提交将一次性创建管理员并保存配置；该操作仅能执行一次。
          </p>
        </section>

        <!-- 底部导航 -->
        <div class="setup-actions">
          <NButton
            v-if="step > 0"
            quaternary
            @click="prev"
          >
            <template #icon>
              <ArrowLeft />
            </template>
            上一步
          </NButton>
          <div class="setup-actions__spacer" />
          <template v-if="step < STEP_TITLES.length - 1">
            <NButton
              v-if="step === 3"
              quaternary
              @click="form.smtpEnabled = false; next()"
            >
              跳过邮件配置
            </NButton>
            <NButton
              type="primary"
              @click="next"
            >
              下一步
              <template #icon>
                <ArrowRight />
              </template>
            </NButton>
          </template>
          <NButton
            v-else
            type="primary"
            size="large"
            :loading="submitting"
            @click="submit"
          >
            <template #icon>
              <Loader2 v-if="submitting" />
              <ShieldCheck v-else />
            </template>
            完成初始化
          </NButton>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.setup-page {
  @apply flex min-h-screen items-center justify-center bg-[var(--prompt-surface-muted)] px-4 py-10;
}

.setup-card {
  @apply w-full max-w-2xl rounded-[var(--prompt-radius-lg)] border border-[var(--prompt-border)] bg-[var(--prompt-surface)] p-6 shadow-sm sm:p-10;
}

.setup-header {
  @apply mb-8 text-center;
}

.setup-brand {
  @apply text-xs font-semibold uppercase tracking-[0.22em] text-[var(--prompt-text-faint)];
}

.setup-title {
  @apply mt-2 text-2xl font-semibold text-[var(--prompt-text)] sm:text-3xl;
}

.setup-subtitle {
  @apply mt-2 text-sm text-[var(--prompt-text-muted)];
}

.setup-steps {
  @apply mb-8 flex flex-wrap items-center gap-x-2 gap-y-3;
}

.setup-step {
  @apply flex items-center gap-2 text-xs text-[var(--prompt-text-faint)];
}

.setup-step__index {
  @apply flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-[var(--prompt-border)] text-[11px] font-semibold;
}

.setup-step--active {
  @apply font-semibold text-[var(--prompt-text)];
}

.setup-step--active .setup-step__index {
  @apply border-transparent text-white;
  background-color: var(--prompt-primary);
  color: var(--prompt-primary-contrast);
}

.setup-step--done .setup-step__index {
  @apply border-transparent;
  background-color: var(--prompt-surface-muted);
  color: var(--prompt-primary);
}

.setup-step__label {
  @apply mr-2;
}

.setup-section {
  @apply min-h-[260px];
}

.setup-checklist {
  @apply flex flex-col gap-3;
}

.setup-check {
  @apply flex items-center gap-4 rounded-2xl border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] px-4 py-3;
}

.setup-check__icon {
  @apply h-5 w-5 shrink-0 text-[var(--prompt-text-muted)];
}

.setup-check__name {
  @apply text-sm font-semibold text-[var(--prompt-text)];
}

.setup-check__value {
  @apply mt-0.5 text-xs text-[var(--prompt-text-muted)];
}

.setup-badge {
  @apply ml-auto rounded-full px-2.5 py-1 text-[11px] font-semibold;
}

.setup-badge--ok {
  @apply bg-emerald-500/10 text-emerald-600;
}

.setup-badge--warn {
  @apply bg-amber-500/10 text-amber-600;
}

.setup-badge--muted {
  @apply bg-[var(--prompt-surface)] text-[var(--prompt-text-faint)];
}

.setup-hint {
  @apply mt-4 rounded-xl bg-amber-500/10 px-4 py-3 text-xs leading-5 text-amber-700;
}

.setup-hint__icon {
  @apply mt-4 h-4 w-4 text-amber-600;
}

.setup-form {
  @apply flex flex-col gap-4;
}

.setup-field {
  @apply flex flex-col gap-1.5;
}

.setup-field__label {
  @apply text-xs font-semibold text-[var(--prompt-text-muted)];
}

.setup-field__hint {
  @apply text-xs text-[var(--prompt-text-faint)];
}

.setup-field--row {
  @apply flex-row items-start gap-3;
}

.setup-grid-2 {
  @apply grid gap-4 sm:grid-cols-2;
}

.setup-test {
  @apply items-center;
}

.setup-test__ok {
  @apply text-xs font-semibold text-emerald-600;
}

.setup-summary {
  @apply flex flex-col gap-3 rounded-2xl border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] p-4;
}

.setup-summary__row {
  @apply flex flex-col gap-1 sm:flex-row sm:gap-4;
}

.setup-summary__row dt {
  @apply w-24 shrink-0 text-xs font-semibold text-[var(--prompt-text-faint)];
}

.setup-summary__row dd {
  @apply text-sm text-[var(--prompt-text)];
}

.setup-actions {
  @apply mt-8 flex items-center gap-3;
}

.setup-actions__spacer {
  @apply flex-1;
}

.setup-link {
  @apply text-sm text-[var(--prompt-text-muted)] underline underline-offset-4;
}

.setup-done {
  @apply flex flex-col items-center py-10 text-center;
}

.setup-done__icon {
  @apply h-14 w-14 text-emerald-500;
}

.setup-done__title {
  @apply mt-4 text-xl font-semibold text-[var(--prompt-text)];
}

.setup-done__desc {
  @apply mt-2 max-w-md text-sm leading-6 text-[var(--prompt-text-muted)];
}

@media (prefers-reduced-motion: reduce) {
  .setup-page,
  .setup-card {
    transition: none;
  }
}
</style>
