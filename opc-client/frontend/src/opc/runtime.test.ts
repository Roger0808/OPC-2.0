import test from 'node:test'
import assert from 'node:assert/strict'
import { relayURL, installTransport, type ClientRuntime } from './runtime'

const config: ClientRuntime = { remoteDesktop: true, publicOrigin: 'https://admin.example.test', transportBase: 'http://127.0.0.1:31000', relayCapability: 'local-capability', profileId: 'test-profile', version: 'test' }

test('only the configured cloud and local API paths are relayed', () => {
  const paths = ['/api/v1/auth/login', '/files?file_path=resource%3A%2F%2Ftest', '/api/v1/knowledge-bases/k/files', '/api/v1/sessions/s/messages/m/files', '/api/v1/embed/e/files']
  for (const path of paths) assert.equal(relayURL(path, config, 'wails://wails.localhost'), config.transportBase + path)
  assert.equal(relayURL('https://evil.test/api/v1/test', config, 'wails://wails.localhost'), null)
  assert.equal(relayURL('/assets/logo.svg', config, 'wails://wails.localhost'), null)
  assert.equal(relayURL('https://admin.example.test/api/v1/test', config, 'wails://wails.localhost'), config.transportBase + '/api/v1/test')
})

test('fetch retains auth scope and abort signal; external requests never get the capability', async () => {
  const calls: Array<{ input: any; init: any }> = []
  class Socket { static OPEN = 1; url: string; constructor(url: any) { this.url = String(url) } }
  ;(globalThis as any).window = {
    __OPC_CLIENT_RUNTIME__: config, location: { origin: 'wails://wails.localhost' }, WebSocket: Socket,
    fetch: async (input: any, init: any) => { calls.push({ input, init }); return new Response('ok') },
  }
  ;(globalThis as any).localStorage = { getItem: () => 'cloud-token' }
  installTransport()
  const signal = new AbortController().signal
  await window.fetch('/api/v1/embed/e/files', { headers: { Authorization: 'Embed guest' }, signal })
  assert.equal(calls[0]!.input, config.transportBase + '/api/v1/embed/e/files')
  assert.equal(calls[0]!.init.headers.get('Authorization'), 'Embed guest')
  assert.equal(calls[0]!.init.headers.get('X-OPC-Client-Session'), config.relayCapability)
  assert.equal(calls[0]!.init.signal, signal)
  assert.equal(calls[0]!.init.credentials, 'omit')
  await window.fetch('https://external.test/image.png')
  assert.equal(calls[1]!.init, undefined)
  await assert.rejects(window.fetch('https://external.test/api', { headers: { Authorization: 'Bearer cloud-token' } }))
  await assert.rejects(window.fetch('https://external.test/api', { headers: { 'X-OPC-Client-Session': 'secret' } }))
  const ws = new window.WebSocket('ws://127.0.0.1:31000/api/v1/sessions/s/sandbox/terminal?ticket=t')
  const url = new URL(ws.url)
  assert.equal(url.searchParams.get('ticket'), 't')
  assert.equal(url.searchParams.get('opc_client_session'), config.relayCapability)
})
