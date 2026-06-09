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
})
