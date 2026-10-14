import test from 'node:test'
import assert from 'node:assert/strict'
import { installStorage, flushSession } from './storage'
import { refreshAccessTokenShared } from '../utils/authRefresh'

class MemoryStorage {
  data = new Map<string, string>()
  get length() { return this.data.size }
  getItem(k: string) { return this.data.get(k) ?? null }
  setItem(k: string, v: string) { this.data.set(k, String(v)) }
  removeItem(k: string) { this.data.delete(k) }
  key(i: number) { return Array.from(this.data.keys())[i] ?? null }
  clear() { this.data.clear() }
}

test('server-scoped storage, secure token batching, transient refresh and logout', async () => {
  const saved: any[] = []
  const local = new MemoryStorage(), session = new MemoryStorage()
  local.setItem('weknora_token', 'original-lite-token')
  local.setItem('opc-client:other:weknora_user', 'other server user')
  const win = new EventTarget() as any
  win.localStorage = local; win.sessionStorage = session
  win.location = { origin: 'wails://wails.localhost', pathname: '/login' }
  win.__OPC_CLIENT_RUNTIME__ = { remoteDesktop: true, profileId: 'test' }
  win.go = { main: { App: {
    GetSavedSession: async () => ({ token: 'secure-token', refreshToken: 'secure-refresh' }),
    SaveSession: async (profile: string, value: any) => { saved.push({ profile, ...value }) },
  } } }
  Object.assign(globalThis, { window: win, Storage: MemoryStorage, localStorage: local, sessionStorage: session })
  await installStorage()
  assert.equal(local.getItem('weknora_token'), 'secure-token')
  assert.equal(local.data.get('weknora_token'), 'original-lite-token')
  local.setItem('weknora_token', 'rotated-token'); local.setItem('weknora_refresh_token', 'rotated-refresh')
  await flushSession()
  assert.deepEqual(saved.at(-1), { profile: 'test', token: 'rotated-token', refreshToken: 'rotated-refresh' })
  assert.equal(local.data.has('opc-client:test:weknora_token'), false)
  local.setItem('weknora_user', 'current user')
  assert.equal(local.data.get('opc-client:test:weknora_user'), 'current user')
  assert.equal(local.getItem('weknora_lite_mode'), null)

  await assert.rejects(refreshAccessTokenShared({ refresh: async () => { throw { status: 502 } } }))
  assert.equal(local.getItem('weknora_token'), 'rotated-token', 'network failure wiped credentials')
  await assert.rejects(refreshAccessTokenShared({ refresh: async () => { throw { status: 401 } } }))
  await flushSession()
  assert.deepEqual(saved.at(-1), { profile: 'test', token: '', refreshToken: '' })
  assert.equal(local.data.get('weknora_token'), 'original-lite-token')
  assert.equal(local.data.get('opc-client:other:weknora_user'), 'other server user')
})
