<script setup lang="ts">
defineProps<{
  columns: number
  rows?: number
}>()
</script>

<template>
  <div class="animate-pulse">
    <div v-for="i in (rows || 5)" :key="i" class="flex space-x-4 py-4 border-b dark:border-white/5">
      <div v-for="j in columns" :key="j" :class="j === 1 ? 'w-1/4' : 'flex-1'" class="h-4 bg-gray-200 dark:bg-gray-700 rounded"></div>
    </div>
  </div>
</template>
