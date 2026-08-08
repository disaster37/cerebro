import { vi } from 'vitest'
import { ref } from 'vue'

const states = new Map<string, any>()

;(globalThis as any).useState = (key: string, init?: () => any) => {
  if (!states.has(key)) states.set(key, ref(init ? init() : undefined))
  return states.get(key)
}

;(globalThis as any).$fetch = vi.fn()
;(globalThis as any).navigateTo = vi.fn()

export function resetTestStubs() {
  states.clear()
}
