// Only same-origin relative paths: blocks open redirects such as //evil.com or /\evil.com
export const safeRedirect = (r) =>
  typeof r === 'string' && /^\/(?![/\\])/.test(r) && r.indexOf('/login') !== 0 ? r : '/'

export const errMsg = (e) =>
  (e && e.response && e.response.data && e.response.data.error) || (e && e.message) || 'Something went wrong'
