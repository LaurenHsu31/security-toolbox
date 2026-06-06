import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Asn1Tree from '../../src/components/Asn1Tree.vue'

const tree = [
  {
    class: 'universal',
    tag: 16,
    tagName: 'SEQUENCE',
    constructed: true,
    length: 14,
    children: [
      {
        class: 'universal',
        tag: 6,
        tagName: 'OBJECT IDENTIFIER',
        constructed: false,
        length: 3,
        value: '2.5.29.14 (subjectKeyIdentifier)'
      },
      {
        class: 'universal',
        tag: 4,
        tagName: 'OCTET STRING',
        constructed: false,
        length: 22,
        hex: '041433'
      }
    ]
  }
]

describe('Asn1Tree', () => {
  it('renders one line per node with tag names and values', () => {
    const w = mount(Asn1Tree, { props: { nodes: tree } })
    const text = w.text()
    expect(text).toContain('SEQUENCE')
    expect(text).toContain('OBJECT IDENTIFIER')
    expect(text).toContain('2.5.29.14 (subjectKeyIdentifier)')
    // a primitive with no interpreted value falls back to its hex
    expect(text).toContain('041433')
    // no generic "Children" / "Constructed" labels from the old dump
    expect(text).not.toContain('Children')
    expect(text).not.toContain('Constructed')
  })

  it('collapses and expands children on click', async () => {
    const w = mount(Asn1Tree, { props: { nodes: tree } })
    expect(w.text()).toContain('OBJECT IDENTIFIER')
    await w.find('.line.clickable').trigger('click')
    expect(w.text()).not.toContain('OBJECT IDENTIFIER')
    await w.find('.line.clickable').trigger('click')
    expect(w.text()).toContain('OBJECT IDENTIFIER')
  })
})
