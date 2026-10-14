let generation = 0
export function scopeGeneration(): number { return generation }
if (typeof window !== 'undefined') {
  window.addEventListener('opc:session-cleared', () => { generation++ })
  window.addEventListener('opc:scope-changed', () => { generation++ })
}
