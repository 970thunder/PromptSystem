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
    message.error('Please choose an image file')
    input.value = ''
    return
  }

  selectedFileName.value = file.name
  uploading.value = true
  try {
    const response = await promptApi.uploadCover(file)
    form.cover = response.data.url
    message.success('Cover uploaded')
  } finally {
    uploading.value = false
    input.value = ''
  }
}

const handleSubmit = async () => {
  syncTags()

  if (!canSubmit.value) {
    message.error('Please fill in the required fields first')
    return
  }

  submitting.value = true
  try {
    const response = isEditing.value
      ? await promptApi.updatePrompt(editingPromptId.value, form)
      : await promptApi.publishPrompt(form)

    promptStore.upsertPrompt(response.data)
    await promptStore.loadHomeFeed()
    message.success(isEditing.value ? 'Prompt updated' : 'Prompt published')
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
            {{ isEditing ? 'Edit Prompt' : 'Publish Prompt' }}
          </div>
          <h1 class="mt-2 text-3xl font-semibold">
            {{ isEditing ? 'Refine and republish your prompt' : 'Publish an image-first AI prompt' }}
          </h1>
        </div>
      </div>

      <div
        v-if="loadingPrompt"
        class="rounded-[28px] border border-black/8 bg-white p-10 text-center text-sm text-[#666666] shadow-[0_16px_40px_rgba(15,23,42,0.06)]"
      >
        Loading prompt...
      </div>

      <div
        v-else
        class="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]"
      >
        <section class="rounded-[28px] border border-black/8 bg-white p-6 shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
          <div class="grid gap-5">
            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">Title</span>
              <input
                v-model="form.title"
                type="text"
                maxlength="80"
                class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                placeholder="Cinematic product poster generator"
              >
            </label>

            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">Description</span>
              <textarea
                v-model="form.description"
                rows="3"
                maxlength="180"
                class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                placeholder="Explain the use case and what kind of output this prompt produces"
              />
            </label>

            <div class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">Cover image</span>
              <label class="flex cursor-pointer flex-col items-center justify-center rounded-[20px] border border-dashed border-black/15 bg-[#faf8f4] px-6 py-8 text-center transition hover:border-black/30">
                <input
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif"
                  class="hidden"
                  @change="handleCoverChange"
                >
                <span class="text-sm text-[#555555]">
                  {{ uploading ? 'Uploading image...' : 'Upload a JPG, PNG, WEBP, or GIF cover' }}
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
                <span class="text-sm font-medium text-[#333333]">Category</span>
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
                <span class="text-sm font-medium text-[#333333]">Model</span>
                <input
                  v-model="form.model"
                  type="text"
                  class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                  placeholder="GPT-4o / Midjourney v6"
                >
              </label>
            </div>

            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">Tags</span>
              <input
                v-model="tagInput"
                type="text"
                class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                placeholder="Comma separated, for example: cinematic, ecommerce, poster"
                @blur="syncTags"
              >
            </label>

            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">Prompt</span>
              <textarea
                v-model="form.content"
                rows="7"
                class="rounded-[18px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm leading-6 outline-none transition focus:border-black/30"
                placeholder="Enter the main prompt body"
              />
            </label>

            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">System prompt</span>
              <textarea
                v-model="form.systemPrompt"
                rows="5"
                class="rounded-[18px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm leading-6 outline-none transition focus:border-black/30"
                placeholder="Enter the role framing or operating constraints"
              />
            </label>

            <div class="grid gap-4 sm:grid-cols-3">
              <label class="grid gap-2">
                <span class="text-sm font-medium text-[#333333]">Temperature</span>
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
                <span class="text-sm font-medium text-[#333333]">Max tokens</span>
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
              Publish checklist
            </div>
            <div class="mt-4 space-y-3 text-sm text-[#444444]">
              <div>1. Use a clean horizontal cover so the gallery layout holds up well.</div>
              <div>2. Let the title describe the result and the description describe the scenario.</div>
              <div>3. Keep tags tight. Three to six usually feels right.</div>
            </div>
          </section>

          <section class="rounded-[28px] border border-black/8 bg-[#111111] p-6 text-white">
            <div class="text-sm text-white/60">
              Storage
            </div>
            <div class="mt-2 text-2xl font-semibold">
              Local first, R2 ready
            </div>
            <p class="mt-3 text-sm leading-6 text-white/70">
              Development defaults to filesystem storage exposed through `/uploads`. Production can switch to Cloudflare R2 while keeping the same upload API.
            </p>

            <button
              class="mt-6 w-full rounded-full bg-white px-4 py-3 text-sm font-medium text-black transition hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="submitting || uploading || !canSubmit"
              @click="handleSubmit"
            >
              {{ submitting ? (isEditing ? 'Saving...' : 'Publishing...') : (isEditing ? 'Save changes' : 'Publish prompt') }}
            </button>
          </section>
        </aside>
      </div>
    </div>
  </div>
</template>
