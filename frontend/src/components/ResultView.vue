<script setup lang="ts">
import Asn1Tree from './Asn1Tree.vue'
import { copyText } from '../clipboard'

defineProps<{ value: unknown; depth?: number }>()

function isObject(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}
function isArray(v: unknown): v is unknown[] {
  return Array.isArray(v)
}
// An array of decoded ASN.1 nodes — render it as a proper tree, not as a
// generic nested key/value dump.
function isAsn1Tree(v: unknown): v is Record<string, unknown>[] {
  return (
    Array.isArray(v) &&
    v.length > 0 &&
    v.every((n) => isObject(n) && 'tagName' in n && 'class' in n && 'constructed' in n)
  )
}
function leafText(v: unknown): string {
  if (v === null) return 'null'
  if (typeof v === 'string') return v
  return JSON.stringify(v)
}
function looksMono(v: unknown): boolean {
  return typeof v === 'string' && /^[0-9a-fA-F:]{12,}$|^0x|^[A-Za-z0-9+/_=-]{24,}$/.test(v)
}
// Color leaves by type so booleans / numbers / hashes stand out instead of
// being a wall of dark text. Returns a CSS class consumed by the styles below.
function leafClass(v: unknown): string {
  if (typeof v === 'boolean') return v ? 'ok' : 'bad'
  if (typeof v === 'number') return 'num'
  if (v === null) return 'nil'
  if (looksMono(v)) return 'mono'
  return ''
}
async function copy(v: unknown) {
  await copyText(leafText(v))
}
function humanKey(k: string): string {
  return k
    .replace(/([A-Z])/g, ' $1')
    .replace(/^./, (c) => c.toUpperCase())
    .trim()
}
</script>

<template>
  <div class="result">
    <template v-if="isObject(value)">
      <div v-for="(v, k) in value" :key="k" class="row" :class="{ nested: isObject(v) || isArray(v) }">
        <div class="key">{{ humanKey(String(k)) }}</div>
        <div class="val">
          <ResultView v-if="isObject(v) || isArray(v)" :value="v" :depth="(depth || 0) + 1" />
          <span
            v-else
            class="leaf"
            :class="leafClass(v)"
            title="Click to copy (or select to copy manually)"
            @click="copy(v)"
            >{{ leafText(v) }}</span
          >
        </div>
      </div>
    </template>

    <Asn1Tree v-else-if="isAsn1Tree(value)" :nodes="value" />

    <template v-else-if="isArray(value)">
      <ul class="list">
        <li v-for="(v, i) in value" :key="i">
          <ResultView v-if="isObject(v) || isArray(v)" :value="v" :depth="(depth || 0) + 1" />
          <span
            v-else
            class="leaf"
            :class="leafClass(v)"
            title="Click to copy (or select to copy manually)"
            @click="copy(v)"
            >{{ leafText(v) }}</span
          >
        </li>
      </ul>
    </template>

    <template v-else>
      <span class="leaf" :class="leafClass(value)" title="Click to copy" @click="copy(value)">{{ leafText(value) }}</span>
    </template>
  </div>
</template>

<style scoped>
.result {
  width: 100%;
}
.row {
  display: grid;
  grid-template-columns: minmax(120px, 200px) minmax(0, 1fr);
  gap: 16px;
  padding: 10px 0;
  border-bottom: 1px solid var(--border);
  align-items: start;
}
.row:last-child {
  border-bottom: none;
}
/* Nested objects/arrays stack vertically and indent by a small fixed amount,
   instead of reserving a full key column at every depth — otherwise deep trees
   (e.g. the ASN.1 dump) blow out horizontally. */
.row.nested {
  display: block;
}
.row.nested > .key {
  margin-bottom: 6px;
}
.row.nested > .val {
  padding-left: 12px;
  border-left: 2px solid var(--border);
}
.key {
  color: var(--label);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: -0.005em;
  padding-top: 2px;
}
.val {
  min-width: 0;
}
.list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.list > li {
  padding: 6px 10px;
  background: var(--surface-2);
  border-radius: 8px;
}
.leaf {
  display: inline-block;
  color: var(--text);
  font-size: 14px;
  text-align: left;
  padding: 2px 4px;
  margin: -2px -4px;
  border-radius: 6px;
  word-break: break-word;
  white-space: pre-wrap;
  transition: background 0.15s ease;
  max-width: 100%;
  /* Allow drag-selecting the value for manual copy, while click still copies. */
  user-select: text;
  -webkit-user-select: text;
  cursor: copy;
}
.leaf:hover {
  background: rgba(0, 113, 227, 0.1);
}
.leaf.mono {
  font-family: var(--mono);
  font-size: 12.5px;
  line-height: 1.5;
  color: var(--accent);
}
.leaf.ok {
  color: var(--ok);
  font-weight: 600;
}
.leaf.bad {
  color: var(--bad);
  font-weight: 600;
}
.leaf.num {
  color: var(--num);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.leaf.nil {
  color: var(--text-2);
  font-style: italic;
}
</style>
