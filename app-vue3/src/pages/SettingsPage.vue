<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { api, errMsg } from '@/api/http'
import { useSessionStore } from '@/stores/session'

interface Settings { storeName: string; receiptFooter: string }

const session = useSessionStore()
const form = reactive<Settings>({ storeName: '', receiptFooter: '' })
const saved = ref(false)
const error = ref('')
const canWrite = computed(() => session.can('settings.write'))

onMounted(async () => {
  Object.assign(form, await api.get<Settings>('/api/settings'))
})

async function save() {
  error.value = ''
  saved.value = false
  try {
    const out = await api.put<Settings>('/api/settings', { ...form })
    session.outletName = out.storeName
    saved.value = true
  } catch (e) {
    error.value = errMsg(e)
  }
}
</script>

<template>
  <h1>Settings <small class="muted">(Vue 3)</small></h1>
  <div class="card">
    <div class="field"><label>Store name</label><input v-model="form.storeName" :disabled="!canWrite" @input="saved = false" /></div>
    <div class="field"><label>Receipt footer</label><input v-model="form.receiptFooter" :disabled="!canWrite" @input="saved = false" /></div>
    <button class="btn" :disabled="!canWrite" @click="save">Save</button>
    <span v-if="saved" class="flash">Saved</span>
    <p v-if="!canWrite" class="muted">Only managers can change settings (also enforced by the API).</p>
    <p v-if="error" class="error">{{ error }}</p>
  </div>
  <p class="muted">
    Rollback test: run <code>document.cookie = "ui=legacy; path=/"</code> in the console and reload.
    Same URL, legacy page, same API and same session.
  </p>
</template>
