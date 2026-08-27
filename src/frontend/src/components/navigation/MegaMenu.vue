<!-- 文件作用：顶部下推式超级菜单面板（处于正常文档流，展开时把下方内容向下推开）。 -->
<script setup lang="ts">
const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  (e: 'navigate'): void
}>()

const groups = [
  {
    title: '发现',
    items: [
      { label: '全部', to: '/search' },
      { label: '最新', to: '/search?sort=latest' },
      { label: '热门', to: '/search?sort=hot' }
    ]
  },
  {
    title: '按能力',
    items: [
      { label: '图像', to: '/search?tab=workflow&keyword=图像' },
      { label: '文案', to: '/search?tab=workflow&keyword=文案' },
      { label: '代码', to: '/search?tab=workflow&keyword=代码' },
      { label: '工作流', to: '/search?tab=workflow' },
      { label: '智能体', to: '/search?tab=agent' }
    ]
  },
  {
    title: '按动作',
    items: [
      { label: '搜索', to: '/search' },
      { label: '发布', to: '/publish' },
      { label: '个人工作台', to: '/profile' }
    ]
  }
]
</script>

<template>
  <div
    class="prompt-collapse"
    :class="{ 'prompt-collapse--open': props.open }"
  >
    <div class="prompt-collapse__inner">
      <nav
        class="border-t"
        :style="{ borderColor: 'var(--prompt-border)' }"
        aria-label="站点导航"
      >
        <div class="grid gap-6 p-6 sm:grid-cols-3">
          <section
            v-for="group in groups"
            :key="group.title"
          >
            <h3 class="mb-3 text-sm font-semibold">
              {{ group.title }}
            </h3>
            <ul class="space-y-2">
              <li
                v-for="item in group.items"
                :key="item.label"
              >
                <RouterLink
                  :to="item.to"
                  class="text-sm"
                  :style="{ color: 'var(--prompt-text-muted)' }"
                  @click="emit('navigate')"
                >
                  {{ item.label }}
                </RouterLink>
              </li>
            </ul>
          </section>
        </div>
      </nav>
    </div>
  </div>
</template>
