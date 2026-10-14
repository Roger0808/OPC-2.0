import { clientRuntime } from '@/opc/runtime'
export function getApiBaseUrl(): string {
  return clientRuntime()?.transportBase || (import.meta.env.BASE_URL || '/').replace(/\/+$/, '')
}
