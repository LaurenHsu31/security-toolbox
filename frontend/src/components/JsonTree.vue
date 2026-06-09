<script setup lang="ts">
import { reactive } from 'vue'
import { copyText } from '../clipboard'

const props = defineProps<{ value: unknown }>()

interface Entry {
  key: string
  val: unknown
}

function isObject(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}
function isContainer(v: unknown): boolean {
  return v !== null && typeof v === 'object'
}
function entriesOf(v: unknown): Entry[] | null {
  if (Array.isArray(v)) return v.map((val, i) => ({ key: String(i), val }))
  if (isObject(v)) return Object.keys(v).map((key) => ({ key, val: v[key] }))
  return null
}

// Per-row collapse state; rows start expanded.
const collapsed = reactive<Record<number, boolean>>({})
const isOpen = (i: number) => collapsed[i] !== true
const toggle = (i: number) => (collapsed[i] = isOpen(i))

function preview(v: unknown): string {
  if (Array.isArray(v)) return `[ ${v.length} ${v.length === 1 ? 'item' : 'items'} ]`
  const n = Object.keys(v as object).length
  return `{ ${n} ${n === 1 ? 'key' : 'keys'} }`
}
function scalarClass(v: unknown): string {
  if (typeof v === 'boolean') return v ? 'ok' : 'bad'
  if (typeof v === 'number') return 'num'
  if (v === null) return 'nil'
  return 'str'
}
function display(v: unknown): string {
  if (v === null) return 'null'
  if (typeof v === 'string') return `"${v}"`
  return String(v)
}
function rawCopy(v: unknown): string {
  return typeof v === 'string' ? v : String(v)
}
async function copyVal(v: unknown) {
  await copyText(rawCopy(v))
}
</script>

<template>
  <span
    v-if="entriesOf(props.value) === null"
    class="leafv"
    :class="scalarClass(props.value)"
    title="Click to copy"
    @click="copyVal(props.value)"
    >{{ display(props.value) }}</span
  >
  <ul v-else class="jtree">
    <li v-for="(e, i) in entriesOf(props.value)!" :key="e.key">
      <div class="jrow" :class="{ clickable: isContainer(e.val) }" @click="isContainer(e.val) && toggle(i)">
        <span class="tw">{{ isContainer(e.val) ? (isOpen(i) ? '▾' : '▸') : '' }}</span>
        <span class="k">{{ e.key }}</span>
        <template v-if="isContainer(e.val)">
          <span v-if="!isOpen(i)" class="preview">{{ preview(e.val) }}</span>
        </template>
        <span
          v-else
          class="leafv"
          :class="scalarClass(e.val)"
          title="Click to copy"
          @click.stop="copyVal(e.val)"
          >{{ display(e.val) }}</span
        >
      </div>
      <JsonTree v-if="isContainer(e.val) && isOpen(i)" :value="e.val" class="kids" />
    </li>
  </ul>
</template>

<style scoped>
.jtree {
  list-style: none;
  margin: 0;
  padding: 0;
}
.kids {
  margin-left: 9px;
  padding-left: 13px;
  border-left: 1px solid var(--border);
}
.jrow {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 3px 7px;
  border-radius: 7px;
  font-family: var(--mono);
  font-size: 12.5px;
  line-height: 1.6;
}
.jrow.clickable {
  cursor: pointer;
}
.jrow.clickable:hover {
  background: rgba(0, 113, 227, 0.07);
}
.tw {
  flex: 0 0 0.9em;
  width: 0.9em;
  color: var(--text-2);
  font-family: var(--sans);
  font-size: 10px;
}
.k {
  flex: 0 0 auto;
  font-weight: 600;
  color: var(--label);
}
.k::after {
  content: ':';
  color: var(--text-2);
  font-weight: 400;
}
.preview {
  color: var(--text-2);
  font-family: var(--sans);
  font-size: 11px;
}
.leafv {
  min-width: 0;
  word-break: break-word;
  white-space: pre-wrap;
  user-select: text;
  cursor: copy;
  border-radius: 4px;
  padding: 0 3px;
  margin: 0 -3px;
}
.leafv:hover {
  background: rgba(0, 113, 227, 0.1);
}
.leafv.str {
  color: #0a84c4;
}
.leafv.num {
  color: var(--num);
  font-weight: 600;
}
.leafv.ok {
  color: var(--ok);
  font-weight: 600;
}
.leafv.bad {
  color: var(--bad);
  font-weight: 600;
}
.leafv.nil {
  color: var(--text-2);
  font-style: italic;
}
</style>
