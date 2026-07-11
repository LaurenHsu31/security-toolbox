import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import JsonTree from '../../src/components/JsonTree.vue'

describe('JsonTree', () => {
  it('renders keys and scalar values', () => {
    const w = mount(JsonTree, { props: { value: { a: 1, nested: { x: true } } } })
    const t = w.text()
    expect(t).toContain('a')
    expect(t).toContain('1')
    expect(t).toContain('nested')
    expect(t).toContain('x')
    expect(t).toContain('true')
  })

  it('collapses and expands a container on click', async () => {
    const w = mount(JsonTree, { props: { value: { nested: { x: 1 } } } })
    expect(w.text()).toContain('x')
    await w.find('.jrow.clickable').trigger('click')
    expect(w.text()).not.toContain('x')
    expect(w.text()).toContain('key') // collapsed preview, e.g. "{ 1 key }"
    await w.find('.jrow.clickable').trigger('click')
    expect(w.text()).toContain('x')
  })

  it('resets collapse state when a new value arrives', async () => {
    const w = mount(JsonTree, { props: { value: { nested: { x: 1 } } } })
    await w.find('.jrow.clickable').trigger('click')
    expect(w.text()).not.toContain('x')
    // A new result must start fully expanded, not inherit row 0's collapse.
    await w.setProps({ value: { other: { y: 2 } } })
    expect(w.text()).toContain('y')
  })

  it('toggles with the keyboard', async () => {
    const w = mount(JsonTree, { props: { value: { nested: { x: 1 } } } })
    const row = w.find('.jrow.clickable')
    expect(row.attributes('role')).toBe('button')
    expect(row.attributes('tabindex')).toBe('0')
    await row.trigger('keydown.enter')
    expect(w.text()).not.toContain('x')
    await row.trigger('keydown.space')
    expect(w.text()).toContain('x')
  })
})
