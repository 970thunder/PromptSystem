<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { promptApi } from '@/api/promptApi'
import { usePromptStore } from '@/stores/prompt'
import type { PublishPromptRequest } from '@/types'
import { resolveMediaUrl } from '@/utils/mediaUrl'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const promptStore = usePromptStore()

const submitting = ref(false)
const uploading = ref(false)
const loadingPrompt = ref(false)
const selectedFileName = ref('')
const tagInput = ref('')
const editingPromptId = computed(() => Number(route.query.edit) || 0)
const isEditing = computed(() => editingPromptId.value > 0)

const form = reactive<PublishPromptRequest>({
  title: '',
  description: '',
  cover: '',
  content: '',
  systemPrompt: '',
  model: 'GPT-4o',
  params: {
    temperature: 0.7,
    topP: 0.9,
    maxTokens: 1200
  },
  categoryId: 1,
  tags: []
})

const canSubmit = computed(() =>
  Boolean(
    form.title.trim()
    && form.description.trim()
    && form.cover.trim()
    && form.content.trim()
    && form.systemPrompt.trim()
    && form.model.trim()
    && form.categoryId > 0
  )
)

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
      form.params = { ...prompt.params }
      form.categoryId = prompt.categoryId
      form.tags = [...prompt.tags]
      tagInput.value = prompt.tags.join(', ')
    } finally {
      loadingPrompt.value = false
    }
  }
})

const syncTags = () => {
  form.tags = tagInput.value
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter(Boolean)
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

const handleSubmit = async () => {
  syncTags()

  if (!canSubmit.value) {
    message.error('请先填写必填项')
    return
  }

  submitting.value = true
  try {
    const response = isEditing.value
      ? await promptApi.updatePrompt(editingPromptId.value, form)
      : await promptApi.publishPrompt(form)

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
  <div class="min-h-screen bg-[#f5f3ee] px-4 py-8 text-[#111111] sm:px-6 lg:px-8">
    <div class="mx-auto max-w-6xl">
      <div class="mb-8 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <div class="text-sm uppercase tracking-[0.2em] text-[#7a7a7a]">
            {{ isEditing ? '编辑提示词' : '发布提示词' }}
          </div>
          <h1 class="mt-2 text-3xl font-semibold">
            {{ isEditing ? '优化并重新发布你的提示词' : '发布以图像为主的 AI 提示词' }}
          </h1>
        </div>
      </div>

      <div
        v-if="loadingPrompt"
        class="rounded-[28px] border border-black/8 bg-white p-10 text-center text-sm text-[#666666] shadow-[0_16px_40px_rgba(15,23,42,0.06)]"
      >
        正在加载提示词...
      </div>

      <div
        v-else
        class="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]"
      >
        <section class="rounded-[28px] border border-black/8 bg-white p-6 shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
          <div class="grid gap-5">
            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">标题</span>
              <input
                v-model="form.title"
                type="text"
                maxlength="80"
                class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                placeholder="例如：电影感产品海报生成器"
              >
            </label>

            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">描述</span>
              <textarea
                v-model="form.description"
                rows="3"
                maxlength="180"
                class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                placeholder="说明使用场景与预期输出类型"
              />
            </label>

            <div class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">封面图</span>
              <label class="flex cursor-pointer flex-col items-center justify-center rounded-[20px] border border-dashed border-black/15 bg-[#faf8f4] px-6 py-8 text-center transition hover:border-black/30">
                <input
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif"
                  class="hidden"
                  @change="handleCoverChange"
                >
                <span class="text-sm text-[#555555]">
                  {{ uploading ? '正在上传图片...' : '上传 JPG、PNG、WEBP 或 GIF 封面' }}
                </span>
                <span
                  v-if="selectedFileName"
                  class="mt-2 text-xs text-[#888888]"
                >
                  {{ selectedFileName }}
                </span>
              </label>

              <div
                v-if="form.cover"
                class="overflow-hidden rounded-[20px] border border-black/8 bg-black"
              >
                <img
                  :src="resolveMediaUrl(form.cover)"
                  alt="cover preview"
                  class="h-[280px] w-full object-cover"
                >
              </div>
            </div>

            <div class="grid gap-5 sm:grid-cols-2">
              <label class="grid gap-2">
                <span class="text-sm font-medium text-[#333333]">分类</span>
                <select
                  v-model="form.categoryId"
                  class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                >
                  <option
                    v-for="category in promptStore.categories"
                    :key="category.id"
                    :value="category.id"
                  >
                    {{ category.name }}
                  </option>
                </select>
              </label>

              <label class="grid gap-2">
                <span class="text-sm font-medium text-[#333333]">模型</span>
                <input
                  v-model="form.model"
                  type="text"
                  class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                  placeholder="GPT-4o / Midjourney v6"
                >
              </label>
            </div>

            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">标签</span>
              <input
                v-model="tagInput"
                type="text"
                class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                placeholder="逗号分隔，例如：电影感, 电商, 海报"
                @blur="syncTags"
              >
            </label>

            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">提示词</span>
              <textarea
                v-model="form.content"
                rows="7"
                class="rounded-[18px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm leading-6 outline-none transition focus:border-black/30"
                placeholder="输入主提示词正文"
              />
            </label>

            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">系统提示词</span>
              <textarea
                v-model="form.systemPrompt"
                rows="5"
                class="rounded-[18px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm leading-6 outline-none transition focus:border-black/30"
                placeholder="输入角色设定或运行约束"
              />
            </label>

            <div class="grid gap-4 sm:grid-cols-3">
              <label class="grid gap-2">
                <span class="text-sm font-medium text-[#333333]">温度</span>
                <input
                  v-model.number="form.params.temperature"
                  type="number"
                  min="0"
                  max="2"
                  step="0.1"
                  class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                >
              </label>

              <label class="grid gap-2">
                <span class="text-sm font-medium text-[#333333]">Top P</span>
                <input
                  v-model.number="form.params.topP"
                  type="number"
                  min="0"
                  max="1"
                  step="0.05"
                  class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                >
              </label>

              <label class="grid gap-2">
                <span class="text-sm font-medium text-[#333333]">最大 Token</span>
                <input
                  v-model.number="form.params.maxTokens"
                  type="number"
                  min="1"
                  step="100"
                  class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                >
              </label>
            </div>
          </div>
        </section>

        <aside class="grid gap-6 self-start">
          <section class="rounded-[28px] border border-black/8 bg-white p-6 shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
            <div class="text-sm text-[#777777]">
              发布清单
            </div>
            <div class="mt-4 space-y-3 text-sm text-[#444444]">
              <div>1. 使用清晰的横版封面，画廊布局效果更好。</div>
              <div>2. 标题描述结果，描述说明使用场景。</div>
              <div>3. 标签保持精简，通常 3–6 个即可。</div>
            </div>
          </section>

          <section class="rounded-[28px] border border-black/8 bg-[#111111] p-6 text-white">
            <div class="text-sm text-white/60">
              存储
            </div>
            <div class="mt-2 text-2xl font-semibold">
              本地优先，可切换 R2
            </div>
            <p class="mt-3 text-sm leading-6 text-white/70">
              开发环境默认通过 `/uploads` 暴露本地文件存储；生产环境可切换至 Cloudflare R2，上传 API 保持一致。
            </p>

            <button
              class="mt-6 w-full rounded-full bg-white px-4 py-3 text-sm font-medium text-black transition hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="submitting || uploading || !canSubmit"
              @click="handleSubmit"
            >
              {{ submitting ? (isEditing ? '保存中...' : '发布中...') : (isEditing ? '保存修改' : '发布提示词') }}
            </button>
          </section>
        </aside>
      </div>
    </div>
  </div>
</template>
