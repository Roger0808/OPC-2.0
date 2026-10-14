import { nativeApp, clientRuntime, relayURL } from './runtime'
import { flushSession } from './storage'

export async function downloadResource(url: string, name: string, headers: Record<string, string> = {}, options: { method?: 'GET' | 'POST'; body?: unknown; signal?: AbortSignal } = {}): Promise<boolean> {
  const app = nativeApp(), c = clientRuntime()
  if (!app || !c) return false
  const relayed = relayURL(url, c, window.location.origin)
  if (!relayed) return false
  const u = new URL(relayed)
  await flushSession()
  const embed = (headers.Authorization || '').startsWith('Embed ') ? headers.Authorization!.slice(6) : ''
  if (options.signal?.aborted) return true
  const abort = () => { void app.CancelDownload() }
  options.signal?.addEventListener('abort', abort, { once: true })
  try {
    await app.DownloadResource(u.pathname + u.search, name, headers['X-Tenant-ID'] || localStorage.getItem('weknora_selected_tenant_id') || '', embed, options.method || 'GET', options.body === undefined ? '' : JSON.stringify(options.body))
  } finally { options.signal?.removeEventListener('abort', abort) }
  return true // Canceling the save dialog still consumes the action.
}

export function installDownloads(): void {
  // Covers existing generated CSV/JSON/Markdown exports without rewriting their producers.
  const handle = (a: HTMLAnchorElement): boolean => {
    if (!a.hasAttribute('download') || !nativeApp()) return false
    const url = a.href, name = a.download || 'download'
    const c = clientRuntime()!
    if (!url.startsWith('blob:') && !url.startsWith('data:') && !relayURL(url, c, window.location.origin)) return false
    // Start reading immediately: some existing callers revoke the blob on nextTick.
    const read = url.startsWith('blob:') || url.startsWith('data:') ? fetch(url).then(r => r.blob()) : null
    void (async () => {
      if (!read && await downloadResource(url, name)) return
      const blob = await read!
      if (blob.size > 64 * 1024 * 1024) throw new Error('文件过大，请使用文件面板中的流式下载')
      const encoded = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader()
        reader.onload = () => resolve(String(reader.result).split(',')[1] || '')
        reader.onerror = () => reject(new Error('读取生成文件失败'))
        reader.readAsDataURL(blob)
      })
      await nativeApp()!.SaveData(name, encoded)
    })().catch(err => window.dispatchEvent(new CustomEvent('opc:download-error', { detail: String(err?.message || err) })))
    return true
  }
  const click = HTMLAnchorElement.prototype.click
  HTMLAnchorElement.prototype.click = function() { if (!handle(this)) click.call(this) }
  document.addEventListener('click', e => {
    const a = (e.target as Element | null)?.closest?.('a')
    if (a instanceof HTMLAnchorElement && handle(a)) { e.preventDefault(); e.stopImmediatePropagation() }
  }, true)
}
