<!-- 文件作用：文档流内的全宽超级菜单。展开时通过 grid-template-rows 真实向下推开下方内容，
     不依赖 position:fixed/absolute，也绝无全屏遮罩。桌面与移动端共用同一份数据。 -->
<script setup lang="ts">
import { watch } from 'vue'

const props = defineProps<{
  open: boolean
  activeSection?: 'home' | 'discover' | 'community' | 'workspace' | null
}>()

const emit = defineEmits<{
  (e: 'navigate'): void
}>()

interface MegaItem {
  label: string
  to?: string
  active?: 'home' | 'discover' | 'community' | 'workspace' | null
  disabled?: boolean
}

interface MegaGroup {
  title: string
  items: MegaItem[]
}

// 桌面与移动端共用的真实链接数据；分组对应手册路由表。
const groups: MegaGroup[] = [
  {
    title: '发现内容',
    items: [
      { label: '发现全部', to: '/search', active: 'discover' },
      { label: '最新', to: '/search?sort=latest', active: 'discover' },
      { label: '热门', to: '/search?sort=popular', active: 'discover' },
      { label: '图像', to: '/search?keyword=图像', active: 'discover' },
      { label: '文案', to: '/search?keyword=文案', active: 'discover' },
      { label: '代码', to: '/search?keyword=代码', active: 'discover' }
    ]
  },
  {
    title: '参与社区',
    items: [
      { label: '工作流', to: '/search?tag=流程', active: 'community' },
      { label: '智能体', to: '/search?tag=智能体', active: 'community' },
      { label: '发布提示词', to: '/publish', active: 'community' }
    ]
  },
  {
    title: '个人工作台',
    items: [
      { label: '我的工作台', to: '/profile', active: 'workspace' }
    ]
  },
  {
    title: '即将开放',
    items: [
      { label: '技能运行器', disabled: true },
      { label: '在线 Playground', disabled: true },
      { label: '创作者学院', disabled: true },
      { label: '提示词交易市场', disabled: true }
    ]
  }
]

// 记录菜单打开/关闭前后主内容区的 top 位移，供验收测量（不输出控制台日志）。
watch(
  () => props.open,
  (isOpen) => {
    const main = document.querySelector('#app main') as HTMLElement | null
    if (!main) {
      return
    }
    const top = Math.round(main.getBoundingClientRect().top)
    document.documentElement.dataset.menuContentTop = String(top)
    document.documentElement.dataset.menuState = isOpen ? 'open' : 'closed'
  }
)
</script>

<template>
  <div
    class="mega"
    :class="{ 'mega--open': open }"
  >
    <div class="mega__inner">
      <nav
        id="mega-menu"
        class="mega__nav"
        :class="{ 'mega__nav--open': open }"
        :inert="!open"
        :aria-hidden="!open"
        aria-label="站点导航"
      >
        <section
          v-for="group in groups"
          :key="group.title"
          class="mega__group"
        >
          <h3 class="mega__title">
            {{ group.title }}
          </h3>
          <ul class="mega__list">
            <li
              v-for="item in group.items"
              :key="item.label"
            >
              <RouterLink
                v-if="!item.disabled"
                :to="item.to ?? '/'"
                class="mega__link"
                :class="{ 'mega__link--active': item.active && item.active === activeSection }"
                @click="emit('navigate')"
              >
                {{ item.label }}
              </RouterLink>
              <span
                v-else
                class="mega__link mega__link--disabled"
                role="link"
                aria-disabled="true"
                :title="`${item.label} 即将开放`"
              >
                {{ item.label }}
                <span class="mega__soon">即将开放</span>
              </span>
            </li>
          </ul>
        </section>
      </nav>
    </div>
  </div>
</template>

<style scoped>
/* 文档流内：grid-template-rows 0fr -> 1fr 实现平滑、真实的下推（非浮层）。 */
.mega {
  display: grid;
  grid-template-rows: 0fr;
  background-color: var(--prompt-bg);
  transition: grid-template-rows var(--prompt-duration-base) var(--prompt-ease-in);
}

.mega--open {
  grid-template-rows: 1fr;
  transition: grid-template-rows var(--prompt-duration-slow) var(--prompt-ease-out);
}

.mega__inner {
  overflow: hidden;
  min-height: 0;
}

.mega__nav {
  display: grid;
  gap: 1.5rem;
  padding: 0 1rem 1.25rem;
  border-top: 1px solid var(--prompt-border);
  margin-top: 0;
  opacity: 0;
  transition: opacity var(--prompt-duration-fast) var(--prompt-ease-out);
}

.mega__nav--open {
  opacity: 1;
  padding-top: 1.25rem;
}

@media (min-width: 640px) {
  .mega__nav {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 1.5rem 2rem;
    padding-left: 1.5rem;
    padding-right: 1.5rem;
  }
}

@media (min-width: 1024px) {
  .mega__nav {
    padding-left: 2rem;
    padding-right: 2rem;
  }
}

.mega__title {
  margin-bottom: 0.75rem;
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--prompt-text-faint);
}

.mega__list {
  display: grid;
  gap: 0.25rem;
}

.mega__link {
  display: block;
  padding: 0.5rem 0.75rem;
  border-radius: var(--prompt-radius-sm);
  font-size: 0.9rem;
  color: var(--prompt-text-muted);
  transition: background-color var(--prompt-duration-fast) var(--prompt-ease-out),
    color var(--prompt-duration-fast) var(--prompt-ease-out);
}

.mega__link:hover {
  background-color: var(--prompt-surface-muted);
  color: var(--prompt-text);
}

.mega__link--active {
  color: var(--prompt-text);
  font-weight: 600;
}

.mega__link--disabled {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  cursor: not-allowed;
  color: var(--prompt-text-faint);
}

.mega__soon {
  font-size: 0.65rem;
  letter-spacing: 0.08em;
  border: 1px solid var(--prompt-border);
  border-radius: 9999px;
  padding: 0.1rem 0.5rem;
  color: var(--prompt-text-faint);
}

@media (prefers-reduced-motion: reduce) {
  .mega,
  .mega--open,
  .mega__nav,
  .mega__link {
    transition-duration: 1ms;
  }
}
</style>
