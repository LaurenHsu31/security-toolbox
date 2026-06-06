import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ResultView from '../../src/components/ResultView.vue'

describe('ResultView', () => {
  it('renders object keys with humanized labels', () => {
    const wrapper = mount(ResultView, {
      props: { value: { commonName: 'vehicle.example', keySize: 256 } }
    })
    const text = wrapper.text()
    expect(text).toContain('Common Name')
    expect(text).toContain('vehicle.example')
    expect(text).toContain('Key Size')
    expect(text).toContain('256')
  })

  it('renders arrays as list items', () => {
    const wrapper = mount(ResultView, {
      props: { value: ['Digital Signature', 'Certificate Sign'] }
    })
    expect(wrapper.findAll('li')).toHaveLength(2)
  })

  it('renders nested structures recursively', () => {
    const wrapper = mount(ResultView, {
      props: { value: { validity: { expired: false } } }
    })
    expect(wrapper.text()).toContain('Expired')
    expect(wrapper.text()).toContain('false')
  })
})
