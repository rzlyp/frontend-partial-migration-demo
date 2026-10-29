<script setup lang="ts">
import { computed } from 'vue'
import { isMigrated } from '@/utils/migrated'

const props = defineProps<{ to: string }>()
// Migrated route -> SPA navigation. Anything else is owned by legacy -> plain <a> (full page load).
const spa = computed(() => isMigrated(props.to))
</script>

<template>
  <RouterLink v-if="spa" :to="to" exact-active-class="active"><slot /></RouterLink>
  <a v-else :href="to"><slot /></a>
</template>
