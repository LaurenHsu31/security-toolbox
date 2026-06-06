import { describe, it, expect, beforeEach } from 'vitest'
import {
  loadFavorites,
  saveFavorites,
  toggleFavorite,
  moveFavorite,
  reconcile
} from '../../src/prefs'

describe('prefs', () => {
  beforeEach(() => localStorage.clear())

  it('round-trips favorites through localStorage', () => {
    expect(loadFavorites()).toEqual([])
    saveFavorites(['cmac', 'hkdf'])
    expect(loadFavorites()).toEqual(['cmac', 'hkdf'])
  })

  it('returns [] for corrupt storage', () => {
    localStorage.setItem('security-toolbox:favorites', 'not json')
    expect(loadFavorites()).toEqual([])
  })

  it('toggles membership without mutating the input', () => {
    const a = ['cmac']
    const b = toggleFavorite(a, 'hkdf')
    expect(b).toEqual(['cmac', 'hkdf'])
    expect(toggleFavorite(b, 'cmac')).toEqual(['hkdf'])
    expect(a).toEqual(['cmac'])
  })

  it('moves items to a new position', () => {
    expect(moveFavorite(['a', 'b', 'c'], 0, 2)).toEqual(['b', 'c', 'a'])
    expect(moveFavorite(['a', 'b', 'c'], 2, 0)).toEqual(['c', 'a', 'b'])
  })

  it('ignores out-of-range or no-op moves', () => {
    const l = ['a', 'b']
    expect(moveFavorite(l, 0, 5)).toBe(l)
    expect(moveFavorite(l, 1, 1)).toBe(l)
  })

  it('reconcile drops unknown names but keeps order', () => {
    expect(reconcile(['x', 'cmac', 'y', 'hkdf'], ['hkdf', 'cmac'])).toEqual(['cmac', 'hkdf'])
  })
})
