<script setup lang="ts">
import { useRoute } from 'vue-router'
const route = useRoute()
</script>

<template>
  <h1>Not migrated yet</h1>
  <div class="card">
    <p>
      <code>{{ route.fullPath }}</code> is not owned by Vue 3. The gateway should have sent it to the legacy app.
      If you see this page, the gateway rules and the manifest are out of sync.
    </p>
    <a class="link" :href="route.fullPath">Try again (full page load)</a>
  </div>
</template>
