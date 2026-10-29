export const state = () => ({ user: null, permissions: [], outlet: { name: '' } })

export const getters = {
  can: (s) => (perm) => s.permissions.indexOf(perm) !== -1,
}

export const mutations = {
  setMe(s, me) { s.user = me.user; s.permissions = me.permissions; s.outlet = me.outlet },
  setStoreName(s, name) { s.outlet = { ...s.outlet, name } },
}
