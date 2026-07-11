<script setup lang="ts">
import { inject, reactive, ref, watch, type Ref } from 'vue'
import { copyText } from '../clipboard'

interface Asn1Node {
  class?: string
  tag?: number
  tagName?: string
  constructed?: boolean
  length?: number
  value?: unknown
  hex?: string
  children?: Asn1Node[]
}

const props = defineProps<{ nodes: Asn1Node[] }>()

// Broadcast from App's "Expand all / Collapse all" buttons; every (recursive)
// tree instance listens. Null when mounted standalone (e.g. unit tests).
type TreeControl = { mode: 'expand' | 'collapse'; seq: number }
const treeControl = inject<Ref<TreeControl | null> | null>('treeControl', null)

// Per-row collapse state; nodes start expanded.
const collapsed = reactive<Record<number, boolean>>({})
const isOpen = (i: number) => collapsed[i] !== true
const toggle = (i: number) => (collapsed[i] = isOpen(i))
function resetCollapsed() {
  for (const k of Object.keys(collapsed)) delete collapsed[Number(k)]
}

// A new node list means a new result: stale collapse state must not carry over.
watch(() => props.nodes, resetCollapsed)

if (treeControl) {
  watch(treeControl, (c) => {
    if (!c) return
    if (c.mode === 'collapse') {
      props.nodes.forEach((n, i) => {
        if (hasKids(n)) collapsed[i] = true
      })
    } else {
      resetCollapsed()
    }
  })
}

function hasKids(n: Asn1Node): boolean {
  return Array.isArray(n.children) && n.children.length > 0
}
function interpreted(n: Asn1Node): boolean {
  return n.value !== null && n.value !== undefined && n.value !== ''
}
function valueText(n: Asn1Node): string {
  return interpreted(n) ? String(n.value) : n.hex || ''
}
function tagClass(n: Asn1Node): string {
  const t = n.tagName || ''
  if (t.includes('OBJECT IDENTIFIER')) return 'oid'
  if (t === 'SEQUENCE' || t === 'SET') return 'seq'
  if (t === 'INTEGER' || t === 'ENUMERATED') return 'int'
  if (t === 'BOOLEAN') return 'bool'
  if (t.includes('String') || t === 'UTF8String') return 'str'
  return ''
}

const copiedIdx = ref<number | null>(null)
let copiedTimer: ReturnType<typeof setTimeout> | undefined
async function copy(text: string, i: number) {
  if (await copyText(text)) {
    copiedIdx.value = i
    clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => (copiedIdx.value = null), 1000)
  }
}
</script>

<template>
  <ul class="tree">
    <li v-for="(n, i) in nodes" :key="i" class="node">
      <div
        class="line"
        :class="{ clickable: hasKids(n) }"
        :role="hasKids(n) ? 'button' : undefined"
        :tabindex="hasKids(n) ? 0 : undefined"
        :aria-expanded="hasKids(n) ? isOpen(i) : undefined"
        @click="hasKids(n) && toggle(i)"
        @keydown.enter.prevent="hasKids(n) && toggle(i)"
        @keydown.space.prevent="hasKids(n) && toggle(i)"
      >
        <span class="tw">{{ hasKids(n) ? (isOpen(i) ? '▾' : '▸') : '' }}</span>
        <span class="tag" :class="tagClass(n)">{{ n.tagName }}</span>
        <span
          v-if="!hasKids(n) && valueText(n)"
          class="val"
          :class="{ hex: !interpreted(n), copied: copiedIdx === i }"
          title="Click to copy"
          @click.stop="copy(valueText(n), i)"
          >{{ valueText(n) }}</span
        >
        <span v-if="hasKids(n)" class="count">{{ n.children!.length }} item{{ n.children!.length === 1 ? '' : 's' }}</span>
        <span class="len">{{ n.length }} B</span>
      </div>
      <Asn1Tree v-if="hasKids(n) && isOpen(i)" :nodes="n.children!" class="kids" />
    </li>
  </ul>
</template>

<style scoped>
.tree {
  list-style: none;
  margin: 0;
  padding: 0;
}
.kids {
  margin-left: 9px;
  padding-left: 13px;
  border-left: 1px solid var(--border);
}
.line {
  display: flex;
  align-items: baseline;
  gap: 9px;
  padding: 3px 7px;
  border-radius: 7px;
  font-family: var(--mono);
  font-size: 12.5px;
  line-height: 1.55;
}
.line.clickable {
  cursor: pointer;
}
.line.clickable:hover,
.line.clickable:focus-visible {
  background: rgba(0, 113, 227, 0.07);
}
.line.clickable:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}
.tw {
  flex: 0 0 0.9em;
  width: 0.9em;
  color: var(--text-2);
  font-family: var(--sans);
  font-size: 10px;
}
.tag {
  flex: 0 0 auto;
  font-weight: 600;
  color: var(--label);
  letter-spacing: -0.01em;
}
.tag.oid {
  color: var(--ok);
}
.tag.seq {
  color: var(--label);
}
.tag.int {
  color: var(--num);
}
.tag.bool {
  color: var(--num);
}
.tag.str {
  color: var(--str);
}
.val {
  min-width: 0;
  color: var(--text);
  word-break: break-all;
  cursor: copy;
  user-select: text;
  -webkit-user-select: text;
  border-radius: 4px;
  padding: 0 3px;
  margin: 0 -3px;
}
.val:hover {
  background: rgba(0, 113, 227, 0.1);
}
.val.copied {
  background: rgba(48, 209, 88, 0.18);
}
.val.copied::after {
  content: ' ✓ Copied';
  font-family: var(--sans);
  font-size: 10px;
  color: var(--ok);
}
.val.hex {
  color: var(--accent);
}
.count {
  flex: 0 0 auto;
  color: var(--text-2);
  font-family: var(--sans);
  font-size: 11px;
}
.len {
  flex: 0 0 auto;
  margin-left: auto;
  color: var(--text-2);
  font-family: var(--sans);
  font-size: 11px;
  white-space: nowrap;
}
</style>
