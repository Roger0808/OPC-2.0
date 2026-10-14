import { clientRuntime, nativeApp, type SavedSession } from './runtime'

const credentialKeys = new Set(['weknora_token', 'weknora_refresh_token'])
let pending: Promise<void> = Promise.resolve()
let scheduled = false
let tokens: SavedSession = { token: '', refreshToken: '' }

function warn(message: string) {
  window.__OPC_CLIENT_STORAGE_WARNING__ = message
  window.dispatchEvent(new CustomEvent('opc:credential-warning', { detail: message }))
}

function scheduleSave() {
  if (scheduled) return
  scheduled = true
  queueMicrotask(() => {
    scheduled = false
    const snapshot = { ...tokens }
    const app = nativeApp()
    if (!app) return // Browser development uses memory-only credentials.
    pending = pending.then(() => app.SaveSession(clientRuntime()!.profileId, snapshot))
      .catch(() => warn('系统凭据存储不可用，本次登录仅在当前会话有效；退出客户端后需重新登录。'))
  })
}

export async function flushSession(): Promise<void> {
  await Promise.resolve() // Let batched token+refreshToken writes enter the queue.
  await pending
}

// Compatibility adapter for the copied frontend's existing synchronous token readers.
// Credentials live in memory and the OS secure store, never WebView localStorage.
// Other keys are scoped to this server; it cannot read the original Lite state.
export async function installStorage(): Promise<void> {
  const c = clientRuntime()!
  const prefix = `opc-client:${c.profileId}:`
  const get = Storage.prototype.getItem
  const set = Storage.prototype.setItem
  const remove = Storage.prototype.removeItem
  const clear = Storage.prototype.clear
  const key = Storage.prototype.key
  const local = window.localStorage
  const session = window.sessionStorage
  const isOurs = (store: Storage) => store === local || store === session

  try { tokens = await nativeApp()?.GetSavedSession() || tokens }
  catch { warn('无法读取系统凭据存储，请重新登录。凭据不会写入明文文件。') }
  Storage.prototype.getItem = function(name: string) {
    if (!isOurs(this)) return get.call(this, name)
    if (name === 'weknora_lite_mode') return null
    if (this === local && credentialKeys.has(name)) return (name === 'weknora_token' ? tokens.token : tokens.refreshToken) || null
    return get.call(this, prefix + name)
  }
  Storage.prototype.setItem = function(name: string, value: string) {
    if (!isOurs(this)) return set.call(this, name, value)
    if (name === 'weknora_lite_mode') return
    if (this === local && credentialKeys.has(name)) {
      const field = name === 'weknora_token' ? 'token' : 'refreshToken'
      tokens[field] = String(value); scheduleSave(); return
    }
    set.call(this, prefix + name, String(value))
  }
  Storage.prototype.removeItem = function(name: string) {
    if (!isOurs(this)) return remove.call(this, name)
    if (this === local && credentialKeys.has(name)) {
      tokens[name === 'weknora_token' ? 'token' : 'refreshToken'] = ''
      scheduleSave()
      window.dispatchEvent(new Event('opc:session-cleared'))
      return
    }
    remove.call(this, prefix + name)
  }
  Storage.prototype.clear = function() {
    if (!isOurs(this)) return clear.call(this)
    const names: string[] = []
    for (let i = 0; i < this.length; i++) { const k = key.call(this, i); if (k?.startsWith(prefix)) names.push(k) }
    names.forEach(k => remove.call(this, k))
    if (this === local) { tokens = { token: '', refreshToken: '' }; scheduleSave(); window.dispatchEvent(new Event('opc:session-cleared')) }
  }
}
