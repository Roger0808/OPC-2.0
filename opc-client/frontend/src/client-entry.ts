import { clientRuntime, installTransport, nativeApp } from './opc/runtime'
import { installStorage } from './opc/storage'
import { installDownloads } from './opc/downloads'

async function start() {
  // Static imports in main.ts construct axios/stores. Set transport and credentials first.
  if (!clientRuntime()) throw new Error('缺少 OPC 客户端运行配置，请使用 opc-client 构建或开发入口')
  // Wails injects bindings at document startup. Do not silently turn a native build into web mode.
  if (['wails', 'wails.localhost'].includes(window.location.hostname)) {
    const deadline = Date.now() + 10000
    while (!nativeApp() && Date.now() < deadline) await new Promise(r => setTimeout(r, 50))
    if (!nativeApp()) throw new Error('客户端初始化失败：原生桥接尚未就绪')
  }
  await installStorage()
  installTransport()
  installDownloads()
  await import('./main')
}
start().catch(err => {
  const el = document.getElementById('app')
  if (el) { el.textContent = String(err?.message || err); el.style.padding = '48px' }
})
