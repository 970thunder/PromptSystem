<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import {
  NButton,
  NCheckbox,
  NInput,
  NInputNumber,
  NSelect,
  NSteps,
  NStep,
  NTab,
  NTabs,
  useMessage
} from 'naive-ui'
import { promptApi } from '@/api/promptApi'
import { usePromptStore } from '@/stores/prompt'
import type { PublishPromptRequest, PromptParams } from '@/types'
import { renderMarkdownPreview } from '@/utils/markdownPreview'
import { resolveMediaUrl } from '@/utils/mediaUrl'

const MAX_TAGS = 30

const route = useRoute()
const router = useRouter()
const message = useMessage()
const promptStore = usePromptStore()

const submitting = ref(false)
const uploading = ref(false)
const loadingPrompt = ref(false)
const selectedFileName = ref('')
const tagInput = ref('')
const currentStep = ref(0)
const useSystemPrompt = ref(false)
const useAdvancedParams = ref(false)
const contentMode = ref<'plain' | 'markdown' | 'json'>('plain')
const jsonError = ref('')

const editingPromptId = computed(() => Number(route.query.edit) || 0)
const isEditing = computed(() => editingPromptId.value > 0)

const wizardSteps = [
  { title: '封面', description: '上传作品封面' },
  { title: '基本信息', description: '标题与分类' },
  { title: '提示词', description: '正文与系统词' },
  { title: '参数与标签', description: '标签与高级项' },
  { title: '确认发布', description: '检查并提交' }
] as const

const defaultParams: PromptParams = {
  temperature: 0.7,
  topP: 0.9,
  maxTokens: 1200
}

const form = reactive<PublishPromptRequest>({
  title: '',
  description: '',
  cover: '',
  content: '',
  systemPrompt: '',
  model: 'Midjourney v6',
  params: { ...defaultParams },
  categoryId: 1,
  tags: []
})

const categoryOptions = computed(() =>
  promptStore.categories.map((category) => ({
    label: category.name,
    value: category.id
  }))
)

const tagCount = computed(() => form.tags.length)

const markdownPreviewHtml = computed(() => renderMarkdownPreview(form.content))

const selectedCategoryName = computed(() =>
  promptStore.categories.find((item) => item.id === form.categoryId)?.name ?? '未选择'
)

const stepValid = computed(() => {
  switch (currentStep.value) {
    case 0:
      return Boolean(form.cover.trim())
    case 1:
      return Boolean(
        form.title.trim()
        && form.description.trim()
        && form.model.trim()
        && form.categoryId > 0
      )
    case 2:
      if (!form.content.trim()) {
        return false
      }
      if (contentMode.value === 'json' && jsonError.value) {
        return false
      }
      return true
    case 3:
      return tagCount.value <= MAX_TAGS
    default:
      return true
  }
})

const canSubmit = computed(() =>
  Boolean(
    form.title.trim()
    && form.description.trim()
    && form.cover.trim()
    && form.content.trim()
    && form.model.trim()
    && form.categoryId > 0
    && tagCount.value <= MAX_TAGS
    && (contentMode.value !== 'json' || !jsonError.value)
  )
)

watch(
  () => form.content,
  (value) => {
    if (contentMode.value !== 'json') {
      jsonError.value = ''
      return
    }
    if (!value.trim()) {
      jsonError.value = ''
      return
    }
    try {
      JSON.parse(value)
      jsonError.value = ''
    } catch {
      jsonError.value = 'JSON 格式无效，请修正或点击格式化'
    }
  }
)

watch(contentMode, (mode) => {
  if (mode !== 'json') {
    jsonError.value = ''
  }
})

onMounted(async () => {
  if (promptStore.categories.length === 0) {
    await promptStore.loadHomeFeed()
  }

  if (isEditing.value) {
    loadingPrompt.value = true
    try {
      const response = await promptApi.getPromptDetail(editingPromptId.value)
      const prompt = response.data
      form.title = prompt.title
      form.description = prompt.description
      form.cover = prompt.cover
      form.content = prompt.content
      form.systemPrompt = prompt.systemPrompt
      form.model = prompt.model
      form.params = { ...defaultParams, ...prompt.params }
      form.categoryId = prompt.categoryId
      form.tags = [...prompt.tags]
      tagInput.value = prompt.tags.join(', ')
      useSystemPrompt.value = Boolean(prompt.systemPrompt.trim())
      useAdvancedParams.value = Boolean(
        prompt.params.temperature !== undefined
        || prompt.params.topP !== undefined
        || prompt.params.maxTokens !== undefined
      )
      detectContentMode(prompt.content)
    } finally {
      loadingPrompt.value = false
    }
  }
})

function detectContentMode(content: string) {
  const trimmed = content.trim()
  if (!trimmed) {
    contentMode.value = 'plain'
    return
  }
  if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
    try {
      JSON.parse(trimmed)
      contentMode.value = 'json'
      return
    } catch {
      // fall through
    }
  }
  if (/^#{1,3}\s|```|\*\*.+\*\*/m.test(trimmed)) {
    contentMode.value = 'markdown'
    return
  }
  contentMode.value = 'plain'
}

const syncTags = () => {
  const parsed = tagInput.value
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter(Boolean)

  const unique: string[] = []
  const seen = new Set<string>()
  for (const tag of parsed) {
    if (seen.has(tag)) {
      continue
    }
    seen.add(tag)
    unique.push(tag)
    if (unique.length >= MAX_TAGS) {
      break
    }
  }

  if (parsed.length > unique.length || unique.length > MAX_TAGS) {
    message.warning(`最多 ${MAX_TAGS} 个标签，已自动截断`)
  }

  form.tags = unique
  tagInput.value = unique.join(', ')
}

const handleCoverChange = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) {
    return
  }

  if (!file.type.startsWith('image/')) {
    message.error('请选择图片文件')
    input.value = ''
    return
  }

  selectedFileName.value = file.name
  uploading.value = true
  try {
    const response = await promptApi.uploadCover(file)
    form.cover = response.data.url
    message.success('封面上传成功')
  } finally {
    uploading.value = false
    input.value = ''
  }
}

const formatJsonContent = () => {
  if (!form.content.trim()) {
    message.warning('请先输入 JSON 内容')
    return
  }
  try {
    form.content = JSON.stringify(JSON.parse(form.content), null, 2)
    jsonError.value = ''
    message.success('JSON 已格式化')
  } catch {
    jsonError.value = 'JSON 格式无效'
    message.error('JSON 格式无效，无法格式化')
  }
}

const goNext = () => {
  if (currentStep.value === 2) {
    if (contentMode.value === 'json' && jsonError.value) {
      message.error('请先修正 JSON 格式')
      return
    }
  }
  if (currentStep.value === 3) {
    syncTags()
  }
  if (!stepValid.value) {
    message.warning('请先完成当前步骤的必填项')
    return
  }
  if (currentStep.value < wizardSteps.length - 1) {
    currentStep.value += 1
  }
}

const goBack = () => {
  if (currentStep.value > 0) {
    currentStep.value -= 1
  }
}

const buildPayload = (): PublishPromptRequest => {
  syncTags()
  const payload: PublishPromptRequest = {
    title: form.title.trim(),
    description: form.description.trim(),
    cover: form.cover.trim(),
    content: form.content.trim(),
    systemPrompt: useSystemPrompt.value ? form.systemPrompt.trim() : '',
    model: form.model.trim(),
    categoryId: form.categoryId,
    tags: [...form.tags],
    params: useAdvancedParams.value
      ? {
        temperature: form.params.temperature ?? defaultParams.temperature,
        topP: form.params.topP ?? defaultParams.topP,
        maxTokens: form.params.maxTokens ?? defaultParams.maxTokens
      }
      : { ...defaultParams }
  }
  return payload
}

const handleSubmit = async () => {
  syncTags()

  if (!canSubmit.value) {
    message.error('请先填写所有必填项')
    return
  }

  if (tagCount.value > MAX_TAGS) {
    message.error(`标签最多 ${MAX_TAGS} 个`)
    return
  }

  submitting.value = true
  try {
    const payload = buildPayload()
    const response = isEditing.value
      ? await promptApi.updatePrompt(editingPromptId.value, payload)
      : await promptApi.publishPrompt(payload)

    promptStore.upsertPrompt(response.data)
    await promptStore.loadHomeFeed()
    message.success(isEditing.value ? '提示词已更新' : '提示词已发布')
    await router.push(`/prompt/${response.data.id}`)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="publish-page">
    <div
      v-if="loadingPrompt"
      class="publish-loading"
    >
      正在加载提示词...
    </div>

    <div
      v-else
      class="publish-layout"
    >
      <section class="publish-cover-pane">
        <div class="publish-cover-pane__header">
          <div>
            <p class="publish-eyebrow">
              {{ isEditing ? '编辑' : '发布' }}
            </p>
            <h1 class="publish-cover-pane__title">
              {{ isEditing ? '编辑提示词' : '发布图像提示词' }}
            </h1>
          </div>
          <RouterLink
            to="/"
            class="publish-back-link"
          >
            返回首页
          </RouterLink>
        </div>

        <div class="publish-cover-pane__body">
          <label
            v-if="!form.cover"
            class="publish-upload-zone"
          >
            <input
              type="file"
              accept="image/png,image/jpeg,image/webp,image/gif"
              class="hidden"
              @change="handleCoverChange"
            >
            <span class="publish-upload-zone__title">
              {{ uploading ? '正在上传...' : '点击上传封面图' }}
            </span>
            <span class="publish-upload-zone__hint">
              JPG · PNG · WEBP · GIF，横版效果更佳
            </span>
            <span
              v-if="selectedFileName"
              class="publish-upload-zone__file"
            >
              {{ selectedFileName }}
            </span>
          </label>

          <div
            v-else
            class="publish-preview"
          >
            <div class="publish-preview__frame">
              <img
                :src="resolveMediaUrl(form.cover)"
                alt="封面预览"
                class="publish-preview__image"
              >
            </div>
            <label class="publish-preview__change">
              <input
                type="file"
                accept="image/png,image/jpeg,image/webp,image/gif"
                class="hidden"
                @change="handleCoverChange"
              >
              {{ uploading ? '上传中...' : '更换封面' }}
            </label>
          </div>

          <p
            v-if="currentStep === 0 && !form.cover"
            class="publish-cover-hint"
          >
            第一步需上传封面，完成后点击「下一步」
          </p>
        </div>
      </section>

      <section class="publish-wizard">
        <header class="publish-wizard__header">
          <NSteps
            :current="currentStep"
            size="small"
          >
            <NStep
              v-for="(step, index) in wizardSteps"
              :key="step.title"
              :title="step.title"
              :description="index === currentStep ? step.description : undefined"
            />
          </NSteps>
        </header>

        <div class="publish-wizard__body">
          <div class="publish-wizard__card panel-card">
            <div
              v-show="currentStep === 0"
              class="publish-step"
            >
              <h2 class="publish-step__title">
                封面
              </h2>
              <p class="publish-step__desc">
                左侧为封面预览区。请上传一张能代表生成效果的横版图片，画廊首页会以封面作为主视觉。
              </p>
              <label class="publish-upload-btn">
                <input
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif"
                  class="hidden"
                  @change="handleCoverChange"
                >
                {{ form.cover ? '重新选择封面' : '从本地上传' }}
              </label>
              <p
                v-if="form.cover"
                class="publish-step__success"
              >
                封面已就绪，可进入下一步
              </p>
            </div>

            <div
              v-show="currentStep === 1"
              class="publish-step publish-step--scroll"
            >
              <h2 class="publish-step__title">
                基本信息
              </h2>

              <label class="publish-field">
                <span class="publish-field__label">标题 <span class="publish-required">*</span></span>
                <NInput
                  v-model:value="form.title"
                  maxlength="80"
                  show-count
                  placeholder="例如：电影感产品海报生成器"
                />
              </label>

              <label class="publish-field">
                <span class="publish-field__label">描述 <span class="publish-required">*</span></span>
                <NInput
                  v-model:value="form.description"
                  type="textarea"
                  :rows="3"
                  maxlength="180"
                  show-count
                  placeholder="说明使用场景、风格与预期输出"
                />
              </label>

              <div class="publish-field-row">
                <label class="publish-field">
                  <span class="publish-field__label">分类 <span class="publish-required">*</span></span>
                  <NSelect
                    v-model:value="form.categoryId"
                    :options="categoryOptions"
                    placeholder="选择图像分类"
                  />
                </label>

                <label class="publish-field">
                  <span class="publish-field__label">适用模型 <span class="publish-required">*</span></span>
                  <NInput
                    v-model:value="form.model"
                    placeholder="Midjourney v6 / SDXL / DALL·E 3"
                  />
                </label>
              </div>
            </div>

            <div
              v-show="currentStep === 2"
              class="publish-step publish-step--fill"
            >
              <div class="publish-step__head">
                <h2 class="publish-step__title">
                  提示词正文
                </h2>
                <NTabs
                  v-model:value="contentMode"
                  type="segment"
                  size="small"
                >
                  <NTab
                    name="plain"
                    tab="纯文本"
                  />
                  <NTab
                    name="markdown"
                    tab="Markdown"
                  />
                  <NTab
                    name="json"
                    tab="JSON"
                  />
                </NTabs>
              </div>

              <div class="publish-content-grid">
                <label class="publish-field publish-field--fill">
                  <span class="publish-field__label">主提示词 <span class="publish-required">*</span></span>
                  <NInput
                    v-model:value="form.content"
                    type="textarea"
                    class="min-h-[200px]"
                    :rows="10"
                    placeholder="输入主提示词；JSON 模式可一键格式化"
                  />
                  <div
                    v-if="contentMode === 'json'"
                    class="flex items-center gap-2"
                  >
                    <NButton
                      size="small"
                      secondary
                      @click="formatJsonContent"
                    >
                      格式化 JSON
                    </NButton>
                    <span
                      v-if="jsonError"
                      class="publish-error"
                    >{{ jsonError }}</span>
                  </div>
                </label>

                <div
                  v-if="contentMode === 'markdown'"
                  class="publish-field publish-field--fill"
                >
                  <span class="publish-field__label">Markdown 预览</span>
                  <div
                    class="publish-preview-box"
                    v-html="markdownPreviewHtml"
                  />
                </div>

                <div
                  v-else-if="contentMode === 'json'"
                  class="publish-field publish-field--fill"
                >
                  <span class="publish-field__label">结构预览</span>
                  <pre class="publish-preview-box publish-preview-box--code">{{ form.content || '输入合法 JSON 后将在此预览' }}</pre>
                </div>
              </div>

              <div class="publish-panel">
                <NCheckbox v-model:checked="useSystemPrompt">
                  使用系统提示词（图像生成网页端通常无需此项）
                </NCheckbox>
                <NInput
                  v-if="useSystemPrompt"
                  v-model:value="form.systemPrompt"
                  class="mt-3"
                  type="textarea"
                  :rows="4"
                  placeholder="角色设定、负面约束、风格统一说明等"
                />
              </div>
            </div>

            <div
              v-show="currentStep === 3"
              class="publish-step publish-step--scroll"
            >
              <h2 class="publish-step__title">
                参数与标签
              </h2>

              <label class="publish-field">
                <span class="publish-field__label">
                  标签
                  <span class="publish-field__hint">{{ tagCount }} / {{ MAX_TAGS }}</span>
                </span>
                <NInput
                  v-model:value="tagInput"
                  placeholder="逗号或换行分隔，例如：电影感, 电商, 海报"
                  @blur="syncTags"
                />
                <p class="publish-field__note">
                  建议 3–8 个精准标签；最多 {{ MAX_TAGS }} 个
                </p>
              </label>

              <div class="publish-panel">
                <NCheckbox v-model:checked="useAdvancedParams">
                  高级参数（温度 / Top P / 最大 Token）
                </NCheckbox>
                <div
                  v-if="useAdvancedParams"
                  class="publish-params-grid"
                >
                  <label class="publish-field">
                    <span class="publish-field__label-muted">温度</span>
                    <NInputNumber
                      v-model:value="form.params.temperature"
                      :min="0"
                      :max="2"
                      :step="0.1"
                      class="w-full"
                    />
                  </label>
                  <label class="publish-field">
                    <span class="publish-field__label-muted">Top P</span>
                    <NInputNumber
                      v-model:value="form.params.topP"
                      :min="0"
                      :max="1"
                      :step="0.05"
                      class="w-full"
                    />
                  </label>
                  <label class="publish-field">
                    <span class="publish-field__label-muted">最大 Token</span>
                    <NInputNumber
                      v-model:value="form.params.maxTokens"
                      :min="1"
                      :step="100"
                      class="w-full"
                    />
                  </label>
                </div>
                <p
                  v-else
                  class="publish-field__note"
                >
                  未开启时将使用默认参数提交
                </p>
              </div>
            </div>

            <div
              v-show="currentStep === 4"
              class="publish-step publish-step--scroll"
            >
              <h2 class="publish-step__title">
                确认发布
              </h2>
              <dl class="publish-confirm">
                <div class="publish-confirm__row">
                  <dt class="publish-confirm__label">
                    标题
                  </dt>
                  <dd class="publish-confirm__value">
                    {{ form.title || '—' }}
                  </dd>
                </div>
                <div class="publish-confirm__row">
                  <dt class="publish-confirm__label">
                    分类
                  </dt>
                  <dd class="publish-confirm__value">
                    {{ selectedCategoryName }}
                  </dd>
                </div>
                <div class="publish-confirm__row">
                  <dt class="publish-confirm__label">
                    模型
                  </dt>
                  <dd class="publish-confirm__value">
                    {{ form.model || '—' }}
                  </dd>
                </div>
                <div class="publish-confirm__row">
                  <dt class="publish-confirm__label">
                    标签
                  </dt>
                  <dd class="publish-confirm__value">
                    {{ form.tags.length ? form.tags.join(' · ') : '无' }}
                  </dd>
                </div>
                <div class="publish-confirm__row">
                  <dt class="publish-confirm__label">
                    系统提示词
                  </dt>
                  <dd class="publish-confirm__value">
                    {{ useSystemPrompt && form.systemPrompt.trim() ? '已填写' : '未使用' }}
                  </dd>
                </div>
                <div class="publish-confirm__row publish-confirm__row--last">
                  <dt class="publish-confirm__label">
                    高级参数
                  </dt>
                  <dd class="publish-confirm__value">
                    {{ useAdvancedParams ? '已自定义' : '默认' }}
                  </dd>
                </div>
              </dl>
              <p class="publish-confirm__note">
                提交后将在社区画廊展示。请确认封面与提示词内容符合平台规范。
              </p>
            </div>
          </div>
        </div>

        <footer class="publish-wizard__footer">
          <NButton
            v-if="currentStep > 0"
            quaternary
            @click="goBack"
          >
            上一步
          </NButton>
          <span v-else />

          <div class="publish-wizard__actions">
            <NButton
              v-if="currentStep < wizardSteps.length - 1"
              type="primary"
              class="publish-primary-btn"
              :disabled="!stepValid || uploading"
              @click="goNext"
            >
              下一步
            </NButton>
            <NButton
              v-else
              type="primary"
              class="publish-primary-btn"
              :loading="submitting"
              :disabled="!canSubmit || uploading"
              @click="handleSubmit"
            >
              {{ isEditing ? '保存修改' : '发布提示词' }}
            </NButton>
          </div>
        </footer>
      </section>
    </div>
  </div>
</template>

<style scoped>
.publish-page {
  @apply h-screen overflow-hidden bg-[#f5f3ee] text-[#111111];
}

.panel-card {
  min-height: 0;
}

.publish-loading {
  @apply flex h-full items-center justify-center text-sm text-[#666666];
}

.publish-layout {
  @apply grid h-full min-h-0 grid-cols-1 lg:grid-cols-2;
}

.publish-cover-pane {
  @apply relative flex min-h-[40vh] flex-col overflow-hidden border-b border-black/10 bg-[#ebe8e1] lg:min-h-0 lg:border-b-0 lg:border-r;
}

.publish-cover-pane__header {
  @apply flex items-center justify-between px-6 py-5 lg:px-8;
}

.publish-eyebrow {
  @apply text-xs uppercase tracking-[0.2em] text-[#7a7a7a];
}

.publish-cover-pane__title {
  @apply mt-1 text-2xl font-semibold;
}

.publish-back-link {
  @apply text-sm text-[#666666] transition hover:text-black;
}

.publish-cover-pane__body {
  @apply flex min-h-0 flex-1 flex-col px-6 pb-6 lg:px-8 lg:pb-8;
}

.publish-upload-zone {
  @apply flex min-h-0 flex-1 cursor-pointer flex-col items-center justify-center rounded-[28px] border border-dashed border-black/10 bg-white/60 text-center transition hover:border-black/20;
}

.publish-upload-zone__title {
  @apply text-base font-medium text-[#333333];
}

.publish-upload-zone__hint {
  @apply mt-2 text-sm text-[#888888];
}

.publish-upload-zone__file {
  @apply mt-2 text-xs text-[#aaaaaa];
}

.publish-preview {
  @apply relative flex min-h-0 flex-1 overflow-hidden rounded-[28px] border border-black/10 bg-black;
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.12);
}

.publish-preview__frame {
  @apply flex h-full min-h-0 w-full items-center justify-center overflow-hidden bg-[#f6f4ef];
}

.publish-preview__image {
  @apply max-h-full max-w-full h-auto w-auto object-contain;
}

.publish-preview__change {
  @apply absolute bottom-4 right-4 cursor-pointer rounded-full bg-white/95 px-4 py-2 text-sm font-medium text-black shadow transition hover:bg-white;
}

.publish-cover-hint {
  @apply mt-4 text-center text-sm text-[#888888];
}

.publish-wizard {
  @apply flex min-h-0 flex-col overflow-hidden;
}

.publish-wizard__header {
  @apply shrink-0 border-b border-black/10 bg-white/80 px-6 py-5 backdrop-blur-sm lg:px-8;
}

.publish-wizard__body {
  @apply min-h-0 flex-1 overflow-hidden px-6 py-5 lg:px-8;
}

.publish-wizard__card {
  @apply flex h-full flex-col overflow-hidden rounded-[24px] border border-black/10 bg-white p-5;
  box-shadow: 0 16px 40px rgba(15, 23, 42, 0.06);
}

.publish-step {
  @apply flex h-full flex-col justify-center gap-4;
}

.publish-step--scroll {
  @apply justify-start overflow-y-auto pr-1;
}

.publish-step--fill {
  @apply min-h-0 overflow-hidden;
}

.publish-step__title {
  @apply text-lg font-semibold;
}

.publish-step__desc {
  @apply text-sm leading-6 text-[#666666];
}

.publish-step__success {
  @apply text-sm text-[#22a06b];
}

.publish-step__head {
  @apply flex items-center justify-between gap-3;
}

.publish-upload-btn {
  @apply inline-flex w-fit cursor-pointer items-center rounded-full bg-[#111111] px-5 py-2.5 text-sm font-medium text-white transition hover:bg-black/80;
}

.publish-field {
  @apply grid gap-2;
}

.publish-field--fill {
  @apply min-h-0;
}

.publish-field__label {
  @apply text-sm font-medium text-[#333333];
}

.publish-field__label-muted {
  @apply text-sm text-[#555555];
}

.publish-field__hint {
  @apply ml-2 text-xs font-normal text-[#888888];
}

.publish-field__note {
  @apply text-xs text-[#888888];
}

.publish-required {
  @apply text-[#c0392b];
}

.publish-field-row {
  @apply grid gap-4 sm:grid-cols-2;
}

.publish-content-grid {
  @apply grid min-h-0 flex-1 gap-4 lg:grid-cols-2;
}

.publish-preview-box {
  @apply min-h-0 flex-1 overflow-y-auto rounded-[16px] border border-black/10 bg-[#faf8f4] p-4 text-sm leading-6 text-[#333333];
}

.publish-preview-box--code {
  @apply overflow-auto text-xs leading-5 text-[#444444];
}

.publish-panel {
  @apply shrink-0 rounded-[16px] border border-black/10 bg-[#faf8f4] p-4;
}

.publish-error {
  @apply text-xs text-[#c0392b];
}

.publish-params-grid {
  @apply mt-4 grid gap-4 sm:grid-cols-3;
}

.publish-confirm {
  @apply grid gap-3 text-sm;
}

.publish-confirm__row {
  @apply flex justify-between gap-4 border-b border-black/5 pb-2;
}

.publish-confirm__row--last {
  @apply border-b-0;
}

.publish-confirm__label {
  @apply text-[#888888];
}

.publish-confirm__value {
  @apply text-right font-medium;
}

.publish-confirm__note {
  @apply text-xs leading-5 text-[#888888];
}

.publish-wizard__footer {
  @apply flex shrink-0 items-center justify-between gap-3 border-t border-black/10 bg-white/80 px-6 py-4 backdrop-blur-sm lg:px-8;
}

.publish-wizard__actions {
  @apply flex items-center gap-3;
}

:deep(.publish-primary-btn.n-button--primary-type) {
  --n-color: #111111;
  --n-color-hover: #333333;
  --n-color-pressed: #000000;
  --n-color-focus: #111111;
  --n-text-color: #ffffff;
  --n-text-color-hover: #ffffff;
  --n-text-color-pressed: #ffffff;
  --n-text-color-focus: #ffffff;
  border-radius: 9999px;
  padding-left: 1.25rem;
  padding-right: 1.25rem;
}

:deep(.n-input .n-input__textarea-el),
:deep(.n-input .n-input__input-el) {
  font-size: 0.875rem;
}
</style>
