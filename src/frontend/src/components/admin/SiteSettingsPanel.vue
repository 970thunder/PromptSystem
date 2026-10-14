<!-- 文件作用：管理员站点设置面板。读写 /admin/settings：站点名称/描述、
     注册开关、SMTP 运行时配置（密码只写不读，留空保持不变），并支持
     用当前生效配置发送测试邮件。 -->
<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { NButton, NCheckbox, NInput } from 'naive-ui'
import PageLoading from '@/components/feedback/PageLoading.vue'
import PageError from '@/components/feedback/PageError.vue'
import { setupApi, type AdminSiteSettings } from '@/api/setupApi'

const message = useMessage()

const loading = ref(true)
const loadFailed = ref(false)
const saving = ref(false)
const testing = ref(false)
const passwordSet = ref(false)
const smtpFromEnv = ref(false)
const testRecipient = ref('')

const form = reactive({
  siteName: '',
  siteDescription: '',
  smtpHost: '',
  smtpPort: '587',
  smtpUser: '',
  smtpPassword: '',
  smtpFrom: '',
  registrationEnabled: true
})

const applySettings = (data: AdminSiteSettings) => {
  form.siteName = data.siteName
  form.siteDescription = data.siteDescription
  form.smtpHost = data.smtpHost
  form.smtpPort = data.smtpPort || '587'
  form.smtpUser = data.smtpUser
  form.smtpFrom = data.smtpFrom
  form.registrationEnabled = data.registrationEnabled
  passwordSet.value = data.smtpPasswordSet
  smtpFromEnv.value = data.smtpFromEnv
}

const load = async () => {
  loading.value = true
  loadFailed.value = false
  try {
    const response = await setupApi.getAdminSettings()
    applySettings(response.data)
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

const save = async () => {
  if (saving.value) {
    return
  }
  saving.value = true
  try {
    const response = await setupApi.updateAdminSettings({
      siteName: form.siteName.trim(),
      siteDescription: form.siteDescription.trim(),
      smtpHost: form.smtpHost.trim(),
      smtpPort: form.smtpPort.trim(),
      smtpUser: form.smtpUser.trim(),
      smtpPassword: form.smtpPassword,
      smtpFrom: form.smtpFrom.trim(),
      registrationEnabled: form.registrationEnabled
    })
    applySettings(response.data)
    form.smtpPassword = ''
    message.success('站点设置已保存')
  } catch {
    // 拦截器已展示错误详情
  } finally {
    saving.value = false
  }
}

const sendTest = async () => {
  const recipient = testRecipient.value.trim()
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(recipient)) {
    message.warning('请填写有效的测试收件邮箱')
    return
  }
  testing.value = true
  try {
    await setupApi.adminTestSMTP(recipient)
    message.success(`测试邮件已发送至 ${recipient}`)
  } catch {
    // 拦截器已展示错误详情
  } finally {
    testing.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <section
    class="site-settings"
    aria-labelledby="site-settings-title"
  >
    <div class="site-settings__head">
      <h2
        id="site-settings-title"
        class="site-settings__title"
      >
        站点设置
      </h2>
      <p class="site-settings__hint">
        保存后立即生效（无需重启）。邮件服务优先使用此处配置，为空时回退到服务器环境变量 SMTP_*。
      </p>
    </div>

    <PageLoading
      v-if="loading"
      label="正在加载站点设置"
      variant="blocks"
    />

    <PageError
      v-else-if="loadFailed"
      kind="error"
      title="站点设置加载失败"
      description="暂时无法读取站点设置，请稍后重试。"
      action-label="重新加载"
      :busy="loading"
      @action="load()"
    />

    <form
      v-else
      class="site-settings__form"
      @submit.prevent="save()"
    >
      <div class="site-settings__group">
        <h3 class="site-settings__group-title">
          站点信息
        </h3>
        <label class="site-settings__field">
          <span class="site-settings__label">站点名称</span>
          <NInput
            v-model:value="form.siteName"
            placeholder="PromptOS"
            :maxlength="60"
          />
        </label>
        <label class="site-settings__field">
          <span class="site-settings__label">站点描述</span>
          <NInput
            v-model:value="form.siteDescription"
            type="textarea"
            :rows="2"
            :maxlength="200"
            placeholder="一句话介绍这个社区"
          />
        </label>
        <div class="site-settings__field site-settings__field--row">
          <NCheckbox v-model:checked="form.registrationEnabled">
            开放新用户注册
          </NCheckbox>
          <span class="site-settings__note">关闭后注册接口返回 REGISTRATION_DISABLED</span>
        </div>
      </div>

      <div class="site-settings__group">
        <h3 class="site-settings__group-title">
          邮件服务（SMTP）
          <span
            class="site-settings__badge"
            :class="smtpFromEnv ? 'site-settings__badge--ok' : 'site-settings__badge--muted'"
          >{{ smtpFromEnv ? '当前可发送' : '未配置' }}</span>
        </h3>
        <p class="site-settings__note">
          {{
            passwordSet
              ? '已保存 SMTP 密码；留空保存将保持原有密码不变。清空服务器与发件人并保存可移除运行时配置、回退环境变量。'
              : '尚未保存 SMTP 密码（可留空，若服务器无需认证）。'
          }}
        </p>
        <div class="site-settings__grid">
          <label class="site-settings__field">
            <span class="site-settings__label">SMTP 服务器</span>
            <NInput
              v-model:value="form.smtpHost"
              placeholder="smtp.example.com"
            />
          </label>
          <label class="site-settings__field">
            <span class="site-settings__label">端口</span>
            <NInput
              v-model:value="form.smtpPort"
              placeholder="587"
            />
          </label>
          <label class="site-settings__field">
            <span class="site-settings__label">用户名</span>
            <NInput
              v-model:value="form.smtpUser"
              placeholder="noreply@example.com"
            />
          </label>
          <label class="site-settings__field">
            <span class="site-settings__label">密码/授权码</span>
            <NInput
              v-model:value="form.smtpPassword"
              type="password"
              show-password-on="click"
              :placeholder="passwordSet ? '已保存，留空保持不变' : '未保存'"
            />
          </label>
        </div>
        <label class="site-settings__field">
          <span class="site-settings__label">发件人邮箱</span>
          <NInput
            v-model:value="form.smtpFrom"
            placeholder="noreply@example.com"
          />
        </label>
      </div>

      <div class="site-settings__group">
        <h3 class="site-settings__group-title">
          测试邮件
        </h3>
        <div class="site-settings__field site-settings__field--row">
          <NInput
            v-model:value="testRecipient"
            class="site-settings__test-input"
            placeholder="接收测试邮件的邮箱"
          />
          <NButton
            :loading="testing"
            :disabled="!smtpFromEnv && !form.smtpHost"
            @click="sendTest"
          >
            发送测试邮件
          </NButton>
        </div>
        <p class="site-settings__note">
          使用保存后的配置发送；若刚修改尚未保存，请先保存再测试。
        </p>
      </div>

      <div class="site-settings__actions">
        <NButton
          type="primary"
          size="large"
          :loading="saving"
          attr-type="submit"
        >
          保存设置
        </NButton>
      </div>
    </form>
  </section>
</template>

<style scoped>
.site-settings {
  @apply flex flex-col gap-6;
}

.site-settings__head {
  @apply flex flex-col gap-1;
}

.site-settings__title {
  @apply text-lg font-semibold text-[var(--prompt-text)];
}

.site-settings__hint {
  @apply text-xs leading-5 text-[var(--prompt-text-muted)];
}

.site-settings__form {
  @apply flex flex-col gap-6;
}

.site-settings__group {
  @apply flex flex-col gap-4 rounded-2xl border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] p-5;
}

.site-settings__group-title {
  @apply flex items-center gap-2 text-sm font-semibold text-[var(--prompt-text)];
}

.site-settings__badge {
  @apply rounded-full px-2 py-0.5 text-[11px] font-semibold;
}

.site-settings__badge--ok {
  @apply bg-emerald-500/10 text-emerald-600;
}

.site-settings__badge--muted {
  @apply bg-[var(--prompt-surface)] text-[var(--prompt-text-faint)];
}

.site-settings__field {
  @apply flex flex-col gap-1.5;
}

.site-settings__field--row {
  @apply flex-row items-center gap-3;
}

.site-settings__label {
  @apply text-xs font-semibold text-[var(--prompt-text-muted)];
}

.site-settings__note {
  @apply text-xs leading-5 text-[var(--prompt-text-faint)];
}

.site-settings__grid {
  @apply grid gap-4 sm:grid-cols-2;
}

.site-settings__test-input {
  @apply max-w-xs;
}

.site-settings__actions {
  @apply flex justify-end;
}
</style>
