<template>
  <div>
    <h1>Settings <small class="muted">(LEGACY version, rollback target)</small></h1>
    <div class="card">
      <div class="field"><label>Store name</label><input v-model="form.storeName" :disabled="!canWrite"></div>
      <div class="field"><label>Receipt footer</label><input v-model="form.receiptFooter" :disabled="!canWrite"></div>
      <button class="btn" :disabled="!canWrite" @click="save">Save</button>
      <span v-if="saved" class="flash">Saved</span>
      <p v-if="!canWrite" class="muted">Only managers can change settings (also enforced by the API).</p>
      <p v-if="error" class="error">{{ error }}</p>
    </div>
  </div>
</template>

<script>
import { errMsg } from '~/utils/auth'

export default {
  async asyncData({ $api }) {
    return { form: await $api.get('/api/settings') }
  },
  data: () => ({ saved: false, error: '' }),
  computed: { canWrite() { return this.$store.getters['session/can']('settings.write') } },
  methods: {
    async save() {
      this.error = ''
      this.saved = false
      try {
        const out = await this.$api.put('/api/settings', this.form)
        this.$store.commit('session/setStoreName', out.storeName)
        this.saved = true
      } catch (e) {
        this.error = errMsg(e)
      }
    },
  },
}
</script>
