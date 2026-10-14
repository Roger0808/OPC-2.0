import { isCloudClient } from '@/opc/runtime'
import { post } from '@/utils/request'

export function dirFromPickerResponse(body: { data?: { dir?: unknown } } | null | undefined): string {
  const dir = body?.data?.dir
  return typeof dir === 'string' ? dir.trim() : ''
}

// Lite's SPA is reverse-proxied; Wails JS bindings are not injected. The
// folder picker is a same-process HTTP call that opens the native dialog.
export async function pickHostProjectDir(): Promise<string> {
  if (isCloudClient()) throw new Error('云端客户端暂不支持访问本机工作目录；远程沙箱由服务端提供')
  const res = await post<{ data?: { dir?: unknown } }>('/api/v1/system/host-project-dir', {}, { timeout: 0 })
  return dirFromPickerResponse(res)
}
