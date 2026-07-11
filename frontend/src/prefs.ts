// User preferences, persisted to localStorage so they never leave the machine
// (the app makes no network calls beyond its own origin). Currently this is the
// ordered list of "favorite" tools the user has pinned to the top of the
// sidebar. All mutating helpers are pure and return new arrays, which keeps them
// trivial to unit-test and plays nicely with Vue reactivity.

const KEY = 'security-toolbox:favorites'
const THEME_KEY = 'security-toolbox:theme'
const LAST_TOOL_KEY = 'security-toolbox:last-tool'

export function loadLastTool(): string {
  try {
    return localStorage.getItem(LAST_TOOL_KEY) ?? ''
  } catch {
    return ''
  }
}

export function saveLastTool(name: string): void {
  try {
    localStorage.setItem(LAST_TOOL_KEY, name)
  } catch {
    /* best-effort */
  }
}

export type Theme = 'auto' | 'light' | 'dark'

export function loadTheme(): Theme {
  try {
    const t = localStorage.getItem(THEME_KEY)
    if (t === 'light' || t === 'dark') return t
  } catch {
    /* storage unavailable — fall back to auto */
  }
  return 'auto'
}

export function saveTheme(t: Theme): void {
  try {
    if (t === 'auto') localStorage.removeItem(THEME_KEY)
    else localStorage.setItem(THEME_KEY, t)
  } catch {
    /* best-effort */
  }
}

export function loadFavorites(): string[] {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return parsed.filter((x): x is string => typeof x === 'string')
    }
  } catch {
    /* corrupt JSON or storage unavailable — fall back to no favorites */
  }
  return []
}

export function saveFavorites(list: string[]): void {
  try {
    localStorage.setItem(KEY, JSON.stringify(list))
  } catch {
    /* storage full or disabled (e.g. private mode) — best-effort, ignore */
  }
}

export function toggleFavorite(list: string[], name: string): string[] {
  return list.includes(name) ? list.filter((n) => n !== name) : [...list, name]
}

export function moveFavorite(list: string[], from: number, to: number): string[] {
  if (from === to || from < 0 || to < 0 || from >= list.length || to >= list.length) {
    return list
  }
  const next = [...list]
  const [item] = next.splice(from, 1)
  next.splice(to, 0, item)
  return next
}

// Drop any saved names that no longer correspond to a real tool, preserving the
// user's chosen order. Run once the tool list is known.
export function reconcile(list: string[], validNames: string[]): string[] {
  const valid = new Set(validNames)
  return list.filter((n) => valid.has(n))
}
