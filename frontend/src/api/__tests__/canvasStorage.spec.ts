import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { scopeStorage } from '../../../../integrations/infinite-canvas/context'

const descriptors = Object.getOwnPropertyDescriptors(Storage.prototype)
beforeEach(() => { localStorage.clear(); sessionStorage.clear() })
afterEach(() => {
  Object.defineProperties(Storage.prototype, descriptors)
  localStorage.clear()
  sessionStorage.clear()
})

describe('embedded canvas storage', () => {
  it('isolates reads, writes, enumeration and clear from host and other users', () => {
    localStorage.setItem('auth_token', 'host-session')
    localStorage.setItem('sub2api:canvas:8:project', 'other-user')
    scopeStorage(7)
    expect(localStorage.getItem('auth_token')).toBeNull()
    localStorage.setItem('project', 'my-project')
    sessionStorage.setItem('draft', 'my-draft')
    expect(localStorage.length).toBe(1)
    expect(localStorage.key(0)).toBe('project')
    expect(localStorage.getItem('project')).toBe('my-project')
    expect(sessionStorage.getItem('draft')).toBe('my-draft')
    localStorage.clear()
    sessionStorage.clear()
    expect(localStorage.length).toBe(0)
    expect(sessionStorage.length).toBe(0)
    const read = descriptors.getItem.value
    expect(read.call(localStorage, 'auth_token')).toBe('host-session')
    expect(read.call(localStorage, 'sub2api:canvas:8:project')).toBe('other-user')
  })
})
