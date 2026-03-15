import '@testing-library/jest-dom'

// Provide a fully-functional localStorage mock.
// Node 23+ injects a native localStorage (--localstorage-file) that lacks .clear(),
// so we override globalThis.localStorage with a standard in-memory implementation.
const createLocalStorageMock = () => {
  let store: Record<string, string> = {}
  return {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => { store[key] = String(value) },
    removeItem: (key: string) => { delete store[key] },
    clear: () => { store = {} },
    get length() { return Object.keys(store).length },
    key: (i: number) => Object.keys(store)[i] ?? null,
  }
}

Object.defineProperty(globalThis, 'localStorage', {
  value: createLocalStorageMock(),
  writable: true,
})

// Reset the mock store before each test
beforeEach(() => {
  localStorage.clear()
})
