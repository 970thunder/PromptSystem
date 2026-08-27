<!-- 文件作用：页面外壳，提供全局站点导航与内容插槽。 -->
<script setup lang="ts">
import SiteHeader from '@/components/navigation/SiteHeader.vue'
</script>

<template>
  <div
    class="min-h-screen"
    :style="{ backgroundColor: 'var(--prompt-bg)', color: 'var(--prompt-text)' }"
  >
    <SiteHeader />
    <main>
      <slot />
    </main>
  </div>
</template>
