export type ClientRuntime = {
  remoteDesktop: true
  publicOrigin: string
  transportBase: string
  relayCapability: string
  profileId: string
  version: string
}
export type SavedSession = { token: string; refreshToken: string }
export type NativeApp = {
  GetConnectionConfig(): Promise<ClientRuntime>
  GetSavedSession(): Promise<SavedSession>
  SaveSession(profile: string, session: SavedSession): Promise<void>
  SetServerURL(origin: string): Promise<void>
  DownloadFile(path: string, name: string, tenant: string, embedToken: string): Promise<boolean>
  DownloadResource(path: string, name: string, tenant: string, embedToken: string, method: string, body: string): Promise<boolean>
  CancelDownload(): Promise<void>
  SaveData(name: string, data: string): Promise<boolean>
}

declare global {
  interface Window {
    __OPC_CLIENT_RUNTIME__?: ClientRuntime
    __OPC_CLIENT_STORAGE_WARNING__?: string
  }
}

export function clientRuntime(): ClientRuntime | undefined {
  return typeof window === 'undefined' ? undefined : window.__OPC_CLIENT_RUNTIME__
}
export function isCloudClient(): boolean { return !!clientRuntime()?.remoteDesktop }
export function nativeApp(): NativeApp | undefined {
  return (window as any).go?.main?.App
}
export function publicOrigin(): string { return clientRuntime()?.publicOrigin || window.location.origin }
export function apiPath(path: string): boolean {
  return path.startsWith('/api/v1/') || path === '/files' || path.startsWith('/files/') || path.startsWith('/mcp/') || path.startsWith('/r/')
}

// Only selected-server APIs are relayed. Third-party resources never get the capability.
export function relayURL(raw: string, config: ClientRuntime, pageOrigin: string): string | null {
  let u: URL
  // URL.origin is "null" for custom schemes, even when passed wails://wails.localhost.
  // Resolve relative API paths against the selected cloud in that case.
  const base = /^https?:\/\//.test(pageOrigin) ? pageOrigin : config.publicOrigin
  try { u = new URL(raw, base) } catch { return null }
  if (u.protocol === 'wails:' && ['wails', 'wails.localhost'].includes(u.hostname) && !/^https?:\/\//.test(pageOrigin) && apiPath(u.pathname)) {
    return config.transportBase + u.pathname + u.search
  }
  if (![config.publicOrigin, config.transportBase, pageOrigin].includes(u.origin) || !apiPath(u.pathname)) return null
  return config.transportBase + u.pathname + u.search
}

export function transportHeaders(url: string): Record<string, string> {
  const c = clientRuntime()
  return c && relayURL(url, c, window.location.origin) ? { 'X-OPC-Client-Session': c.relayCapability } : {}
}

export function websocketBase(): string {
  const c = clientRuntime()
  if (c) return c.transportBase.replace(/^http/, 'ws')
  return `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}`
}

export function installTransport(): void {
  const c = clientRuntime()
  if (!c) throw new Error('客户端连接配置尚未就绪')
  const originalFetch = window.fetch.bind(window)
  window.fetch = (input: RequestInfo | URL, init?: RequestInit) => {
    const raw = input instanceof Request ? input.url : String(input)
    const target = relayURL(raw, c, window.location.origin)
    const request = input instanceof Request ? input : undefined
    const headers = new Headers(init?.headers ?? request?.headers)
    if (!target) {
      const token = localStorage.getItem('weknora_token')
      if (headers.has('X-OPC-Client-Session') || (token && headers.get('Authorization') === `Bearer ${token}`)) {
        return Promise.reject(new Error('拒绝向其他地址发送客户端凭据'))
      }
      return originalFetch(input, init)
    }
    headers.set('X-OPC-Client-Session', c.relayCapability)
    // Keep Request bodies/signals and per-scope Bearer or Embed headers intact.
    const options: RequestInit = request ? {
      method: request.method, body: ['GET', 'HEAD'].includes(request.method) ? undefined : request.clone().body,
      signal: request.signal, ...init, headers, credentials: 'omit',
    } : { ...init, headers, credentials: 'omit' }
    return originalFetch(target, options)
  }
  const OriginalWebSocket = window.WebSocket
  const config = c
  class ClientWebSocket extends OriginalWebSocket {
    constructor(url: string | URL, protocols?: string | string[]) {
      const u = new URL(String(url))
      const localWS = config.transportBase.replace(/^http/, 'ws')
      const cloudWS = config.publicOrigin.replace(/^http/, 'ws')
      if (apiPath(u.pathname) && [localWS, cloudWS].includes(u.origin)) {
        u.protocol = localWS.startsWith('wss:') ? 'wss:' : 'ws:'
        u.host = new URL(localWS).host
        u.searchParams.set('opc_client_session', config.relayCapability)
      }
      super(u, protocols)
    }
  }
  window.WebSocket = ClientWebSocket
}
