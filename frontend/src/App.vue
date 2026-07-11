<script setup lang="ts">
import { computed, onMounted, onUnmounted, provide, reactive, ref, watch } from 'vue'
import { listTools, runTool, type ToolMeta } from './api'
import { toolUI } from './toolsMeta'
import {
  loadFavorites, saveFavorites, toggleFavorite, moveFavorite, reconcile,
  loadTheme, saveTheme, loadLastTool, saveLastTool, type Theme
} from './prefs'
import { copyText } from './clipboard'
import ResultView from './components/ResultView.vue'
import JsonTree from './components/JsonTree.vue'
import Logo from './components/Logo.vue'

const tools = ref<ToolMeta[]>([])
const activeName = ref<string>('')
const inputValue = ref<string>('')
const controlValues = reactive<Record<string, string>>({})
const revealed = reactive<Record<string, boolean>>({})
const result = ref<unknown>(null)
const errorMsg = ref<string>('')
const loading = ref<boolean>(false)
const loadError = ref<string>('')
const favorites = ref<string[]>(loadFavorites())
const dragIndex = ref<number | null>(null)
const query = ref<string>('')
const searchBox = ref<HTMLInputElement | null>(null)

const activeTool = computed(() => tools.value.find((t) => t.name === activeName.value))
const ui = computed(() => toolUI[activeName.value])

const grouped = computed(() => {
  const order: string[] = []
  const map: Record<string, ToolMeta[]> = {}
  for (const t of tools.value) {
    if (!map[t.category]) {
      map[t.category] = []
      order.push(t.category)
    }
    map[t.category].push(t)
  }
  return order.map((category) => ({ category, items: map[category] }))
})

const filteredGrouped = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return grouped.value
  return grouped.value
    .map((g) => ({
      category: g.category,
      items: g.items.filter(
        (t) =>
          t.title.toLowerCase().includes(q) ||
          t.name.toLowerCase().includes(q) ||
          t.description.toLowerCase().includes(q)
      )
    }))
    .filter((g) => g.items.length > 0)
})

const favoriteTools = computed(() =>
  favorites.value
    .map((name) => tools.value.find((t) => t.name === name))
    .filter((t): t is ToolMeta => Boolean(t))
)

function isFavorite(name: string) {
  return favorites.value.includes(name)
}
function toggleFav(name: string) {
  favorites.value = toggleFavorite(favorites.value, name)
}
function onDrop(to: number) {
  if (dragIndex.value !== null) {
    favorites.value = moveFavorite(favorites.value, dragIndex.value, to)
  }
  dragIndex.value = null
}

watch(favorites, (v) => saveFavorites(v))

// Manual theme override; 'auto' follows the OS via prefers-color-scheme.
const theme = ref<Theme>(loadTheme())
const themeLabel = computed(
  () => ({ auto: '◐ Auto', light: '☀ Light', dark: '☾ Dark' })[theme.value]
)
function cycleTheme() {
  theme.value = theme.value === 'auto' ? 'light' : theme.value === 'light' ? 'dark' : 'auto'
}
watch(
  theme,
  (t) => {
    saveTheme(t)
    if (t === 'auto') delete document.documentElement.dataset.theme
    else document.documentElement.dataset.theme = t
  },
  { immediate: true }
)

// "Expand all / Collapse all" broadcast consumed by JsonTree / Asn1Tree.
const treeControl = ref<{ mode: 'expand' | 'collapse'; seq: number } | null>(null)
provide('treeControl', treeControl)
function setAllTrees(mode: 'expand' | 'collapse') {
  treeControl.value = { mode, seq: (treeControl.value?.seq ?? 0) + 1 }
}

const monoOutput = computed(() => {
  const key = ui.value?.monoOutputKey
  if (key && result.value && typeof result.value === 'object') {
    const v = (result.value as Record<string, unknown>)[key]
    if (typeof v === 'string') return v
  }
  return null
})

// When a result carries a `parsed` value (the JSON formatter in beautify mode),
// render it as a collapsible tree instead of a flat string.
const hasJsonTree = computed(
  () => !!result.value && typeof result.value === 'object' && 'parsed' in (result.value as object)
)
const jsonTreeValue = computed(() =>
  hasJsonTree.value ? (result.value as Record<string, unknown>).parsed : undefined
)

const hasResult = computed(() => !!result.value && !errorMsg.value)
const showTreeButtons = computed(() => hasResult.value && (hasJsonTree.value || monoOutput.value === null))

// Keep what the user typed per tool, so switching to look something up and
// back doesn't lose the pasted input. The result re-runs from the restored
// input via the debounced watcher.
const savedInputs: Record<string, { input: string; controls: Record<string, string> }> = {}

function selectTool(name: string) {
  if (name === activeName.value) return
  if (activeName.value) {
    savedInputs[activeName.value] = { input: inputValue.value, controls: { ...controlValues } }
  }
  activeName.value = name
  result.value = null
  errorMsg.value = ''
  for (const k of Object.keys(controlValues)) delete controlValues[k]
  const spec = toolUI[name]
  spec?.controls?.forEach((c) => {
    controlValues[c.key] = c.default ?? ''
  })
  const saved = savedInputs[name]
  if (saved) Object.assign(controlValues, saved.controls)
  inputValue.value = saved?.input ?? ''
}

function fillSample() {
  if (!ui.value?.sample) return
  // Some samples only decode with matching control values (e.g. the JWT
  // sample's HMAC secret, the ECDH sample's peer key).
  if (ui.value.sampleControls) Object.assign(controlValues, ui.value.sampleControls)
  inputValue.value = ui.value.sample
}

function clearInput() {
  inputValue.value = ''
  result.value = null
  errorMsg.value = ''
}

// Dropping a file onto the input card loads it: text files as-is, binary
// files (e.g. DER certificates) as Base64, which every decoder accepts.
const dragOver = ref(false)
async function onFileDrop(e: DragEvent) {
  dragOver.value = false
  const f = e.dataTransfer?.files?.[0]
  if (!f) return
  if (f.size > 2 * 1024 * 1024) {
    errorMsg.value = 'Dropped file is larger than 2 MB.'
    return
  }
  const buf = new Uint8Array(await f.arrayBuffer())
  let text: string | null = null
  try {
    text = new TextDecoder('utf-8', { fatal: true }).decode(buf)
  } catch {
    /* binary */
  }
  // Control characters mean "binary that happens to be valid UTF-8".
  if (text !== null && !/[\x00-\x08\x0e-\x1f]/.test(text)) {
    inputValue.value = text.trim()
  } else {
    let bin = ''
    for (let i = 0; i < buf.length; i += 0x8000) {
      bin += String.fromCharCode(...buf.subarray(i, i + 0x8000))
    }
    inputValue.value = btoa(bin)
  }
}

async function run() {
  if (!inputValue.value.trim() || !ui.value) {
    result.value = null
    errorMsg.value = ''
    return
  }
  loading.value = true
  errorMsg.value = ''
  const body: Record<string, unknown> = { [ui.value.inputKey]: inputValue.value }
  ui.value.controls?.forEach((c) => {
    body[c.key] = controlValues[c.key]
  })
  try {
    result.value = await runTool(activeName.value, body)
  } catch (e) {
    result.value = null
    errorMsg.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

let timer: ReturnType<typeof setTimeout> | undefined
function scheduleRun() {
  clearTimeout(timer)
  timer = setTimeout(run, 350)
}

watch([inputValue, controlValues], scheduleRun, { deep: true })

const copied = ref(false)
function resultText(): string {
  const r = result.value
  if (r && typeof r === 'object') {
    const f = (r as Record<string, unknown>).formatted
    if (typeof f === 'string') return f
  }
  return monoOutput.value ?? JSON.stringify(r, null, 2)
}
async function copyAll() {
  const ok = await copyText(resultText())
  if (ok) {
    copied.value = true
    setTimeout(() => (copied.value = false), 1200)
  }
}

// "/" or Cmd/Ctrl+K focuses the sidebar search from anywhere (except while
// already typing in a field).
function onKeydown(e: KeyboardEvent) {
  const el = document.activeElement as HTMLElement | null
  const typing =
    !!el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.tagName === 'SELECT' || el.isContentEditable)
  if ((e.key === '/' && !typing) || ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k')) {
    e.preventDefault()
    searchBox.value?.focus()
    searchBox.value?.select()
  }
}

onMounted(async () => {
  window.addEventListener('keydown', onKeydown)
  try {
    tools.value = await listTools()
    favorites.value = reconcile(favorites.value, tools.value.map((t) => t.name))
    const last = loadLastTool()
    const start = tools.value.find((t) => t.name === last) ?? tools.value[0]
    if (start) selectTool(start.name)
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : String(e)
  }
})
onUnmounted(() => window.removeEventListener('keydown', onKeydown))

watch(activeName, (n) => {
  if (n) saveLastTool(n)
})
</script>

<template>
  <div class="shell">
    <header class="topbar">
      <div class="brand">
        <Logo :size="22" />
        <span class="name">security-toolbox</span>
      </div>
      <div class="topbar-right">
        <span class="pill" title="The page is locked to its own origin by Content-Security-Policy.">
          Runs entirely on your machine · nothing is uploaded
        </span>
        <button class="ghost theme-btn" :title="`Theme: ${theme}`" @click="cycleTheme">{{ themeLabel }}</button>
      </div>
    </header>

    <div class="body">
      <aside class="sidebar">
        <div v-if="loadError" class="load-error">
          Could not reach the backend.<br /><small>{{ loadError }}</small>
        </div>

        <div class="search-wrap">
          <input
            ref="searchBox"
            v-model="query"
            class="search"
            type="search"
            placeholder="Filter tools ( / )"
            spellcheck="false"
            autocomplete="off"
            aria-label="Filter tools"
          />
        </div>

        <nav v-if="favoriteTools.length && !query.trim()" class="group">
          <div class="group-title">Favorites</div>
          <div
            v-for="(t, i) in favoriteTools"
            :key="t.name"
            class="nav-row"
            :class="{ active: t.name === activeName, dragging: dragIndex === i }"
            draggable="true"
            @dragstart="dragIndex = i"
            @dragover.prevent
            @drop="onDrop(i)"
            @dragend="dragIndex = null"
          >
            <span class="grip" aria-hidden="true" title="Drag to reorder">⠿</span>
            <button class="nav-item" @click="selectTool(t.name)">{{ t.title }}</button>
            <button
              class="star on"
              aria-label="Remove from favorites"
              title="Remove from favorites"
              @click.stop="toggleFav(t.name)"
            >★</button>
          </div>
        </nav>

        <nav v-for="group in filteredGrouped" :key="group.category" class="group">
          <div class="group-title">{{ group.category }}</div>
          <div
            v-for="t in group.items"
            :key="t.name"
            class="nav-row"
            :class="{ active: t.name === activeName }"
          >
            <button class="nav-item" @click="selectTool(t.name)">{{ t.title }}</button>
            <button
              class="star"
              :class="{ on: isFavorite(t.name) }"
              :aria-label="isFavorite(t.name) ? 'Remove from favorites' : 'Add to favorites'"
              :title="isFavorite(t.name) ? 'Remove from favorites' : 'Add to favorites'"
              @click.stop="toggleFav(t.name)"
            >{{ isFavorite(t.name) ? '★' : '☆' }}</button>
          </div>
        </nav>

        <div v-if="query.trim() && !filteredGrouped.length" class="no-match">
          No tools match “{{ query.trim() }}”.
        </div>
      </aside>

      <main class="content">
        <section v-if="activeTool" class="panel">
          <div class="panel-head">
            <h1>{{ activeTool.title }}</h1>
            <p>{{ activeTool.description }}</p>
          </div>

          <div
            class="card input-card"
            :class="{ 'drag-over': dragOver }"
            @dragover.prevent="dragOver = true"
            @dragleave="dragOver = false"
            @drop.prevent="onFileDrop"
          >
            <div class="label-row">
              <label>{{ ui?.inputLabel }}</label>
              <div class="label-actions">
                <button v-if="inputValue" class="ghost subtle" @click="clearInput">Clear</button>
                <button v-if="ui?.sample" class="ghost" @click="fillSample">Use sample</button>
              </div>
            </div>
            <textarea
              v-model="inputValue"
              :placeholder="ui?.placeholder + '\n(or drop a file here — binary becomes Base64)'"
              spellcheck="false"
              autocapitalize="off"
              autocomplete="off"
            ></textarea>

            <div v-if="ui?.controls?.length" class="controls">
              <div v-for="c in ui.controls" :key="c.key" class="control">
                <label>{{ c.label }}</label>
                <select v-if="c.type === 'select'" v-model="controlValues[c.key]">
                  <option v-for="o in c.options" :key="o.value" :value="o.value">{{ o.label }}</option>
                </select>
                <div v-else-if="c.type === 'password'" class="pw">
                  <input
                    v-model="controlValues[c.key]"
                    :type="revealed[c.key] ? 'text' : 'password'"
                    :placeholder="c.placeholder"
                    spellcheck="false"
                    autocomplete="off"
                  />
                  <button
                    type="button"
                    class="pw-toggle"
                    :aria-label="revealed[c.key] ? 'Hide value' : 'Show value'"
                    @click="revealed[c.key] = !revealed[c.key]"
                  >{{ revealed[c.key] ? 'Hide' : 'Show' }}</button>
                </div>
                <input
                  v-else
                  v-model="controlValues[c.key]"
                  :placeholder="c.placeholder"
                  spellcheck="false"
                  autocomplete="off"
                />
              </div>
            </div>
          </div>

          <div class="result-head">
            <span class="result-title">Result</span>
            <span v-if="loading" class="status">decoding…</span>
            <div class="result-actions">
              <template v-if="showTreeButtons">
                <button class="ghost" @click="setAllTrees('expand')">Expand all</button>
                <button class="ghost" @click="setAllTrees('collapse')">Collapse all</button>
              </template>
              <button
                class="ghost"
                :class="{ invisible: !hasResult }"
                :aria-hidden="!hasResult"
                :tabindex="hasResult ? 0 : -1"
                @click="copyAll"
              >{{ copied ? 'Copied!' : 'Copy all' }}</button>
            </div>
          </div>

          <div v-if="errorMsg" class="card error">{{ errorMsg }}</div>

          <div v-else-if="hasJsonTree" class="card">
            <JsonTree :value="jsonTreeValue" />
          </div>

          <div v-else-if="monoOutput !== null" class="card">
            <pre class="code">{{ monoOutput }}</pre>
          </div>

          <div v-else-if="result" class="card">
            <ResultView :value="result" />
          </div>

          <div v-else class="empty">Paste input above to decode.</div>
        </section>
      </main>
    </div>
  </div>
</template>

<style scoped>
.shell {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.topbar {
  height: 52px;
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 20px;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  backdrop-filter: saturate(180%) blur(20px);
}
.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 16px;
  white-space: nowrap;
}
.topbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.pill {
  font-size: 12px;
  color: var(--text-2);
  background: var(--surface-2);
  border: 1px solid var(--border);
  padding: 5px 12px;
  border-radius: 980px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.theme-btn {
  white-space: nowrap;
  color: var(--text-2);
}
.body {
  flex: 1 1 auto;
  display: flex;
  min-height: 0;
}
.sidebar {
  width: 240px;
  flex: 0 0 auto;
  overflow-y: auto;
  padding: 16px 12px;
  border-right: 1px solid var(--border);
  background: var(--surface-2);
}
.search-wrap {
  padding: 0 2px 14px;
}
.search {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface);
  color: var(--text);
  padding: 7px 10px;
  font-size: 13px;
  outline: none;
}
.search:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(0, 113, 227, 0.2);
}
.no-match {
  font-size: 12.5px;
  color: var(--text-2);
  padding: 4px 10px;
}
.load-error {
  font-size: 12px;
  color: var(--bad);
  padding: 8px 10px;
  margin-bottom: 12px;
}
.group {
  margin-bottom: 18px;
}
.group-title {
  font-size: 11px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-2);
  padding: 0 10px 6px;
}
.nav-row {
  display: flex;
  align-items: center;
  border-radius: 8px;
  transition: background 0.15s ease;
}
.nav-row:hover {
  background: rgba(0, 113, 227, 0.08);
}
.nav-row.active {
  background: var(--accent);
}
.nav-row.active .nav-item {
  color: #fff;
}
.nav-row.dragging {
  opacity: 0.45;
}
.nav-item {
  flex: 1;
  min-width: 0;
  text-align: left;
  border: none;
  background: transparent;
  color: var(--text);
  padding: 8px 10px;
  border-radius: 8px;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.grip {
  padding-left: 6px;
  color: var(--text-2);
  font-size: 12px;
  cursor: grab;
  user-select: none;
}
.nav-row.active .grip {
  color: rgba(255, 255, 255, 0.85);
}
.star {
  flex: 0 0 auto;
  border: none;
  background: transparent;
  color: var(--text-2);
  font-size: 14px;
  line-height: 1;
  padding: 4px 8px;
  opacity: 0;
  transition: opacity 0.12s ease, color 0.12s ease;
}
.nav-row:hover .star,
.star:focus-visible,
.star.on {
  opacity: 1;
}
.star.on {
  color: #f5a623;
}
.nav-row.active .star {
  color: #fff;
}
.content {
  flex: 1 1 auto;
  overflow-y: auto;
  padding: 32px;
}
.panel {
  max-width: 820px;
  margin: 0 auto;
}
.panel-head h1 {
  font-size: 28px;
  font-weight: 600;
  margin: 0 0 6px;
  letter-spacing: -0.01em;
}
.panel-head p {
  margin: 0 0 24px;
  color: var(--text-2);
  font-size: 15px;
  line-height: 1.5;
}
.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 18px;
  box-shadow: var(--shadow);
}
.input-card {
  margin-bottom: 24px;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}
.input-card.drag-over {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(0, 113, 227, 0.2);
}
.label-actions {
  display: flex;
  gap: 8px;
}
.ghost.subtle {
  color: var(--text-2);
}
.label-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.label-row label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-2);
}
textarea {
  width: 100%;
  min-height: 130px;
  resize: vertical;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface-2);
  color: var(--text);
  padding: 12px 14px;
  font-family: var(--mono);
  font-size: 13px;
  line-height: 1.5;
  outline: none;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}
textarea:focus,
input:focus,
select:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(0, 113, 227, 0.2);
}
.controls {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 16px;
}
.control {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 160px;
  flex: 1;
}
.control label {
  font-size: 12px;
  color: var(--text-2);
}
select,
input {
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface-2);
  color: var(--text);
  padding: 9px 11px;
  font-size: 14px;
  outline: none;
}
.pw {
  display: flex;
  gap: 8px;
  align-items: center;
}
.pw input {
  flex: 1;
  min-width: 0;
}
.pw-toggle {
  flex: 0 0 auto;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text-2);
  font-size: 12px;
  padding: 6px 10px;
  border-radius: 8px;
}
.result-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  min-height: 30px;
}
.result-title {
  font-size: 18px;
  font-weight: 600;
}
.status {
  font-size: 13px;
  color: var(--text-2);
}
.result-actions {
  margin-left: auto;
  display: flex;
  gap: 8px;
}
.ghost {
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--accent);
  font-size: 13px;
  padding: 5px 12px;
  border-radius: 980px;
  transition: background 0.15s ease;
}
.ghost:hover {
  background: rgba(0, 113, 227, 0.08);
}
/* Reserve the space so the row doesn't shift when the button appears. */
.ghost.invisible {
  visibility: hidden;
  pointer-events: none;
}
.error {
  color: var(--bad);
  font-family: var(--mono);
  font-size: 13px;
  white-space: pre-wrap;
}
.code {
  margin: 0;
  font-family: var(--mono);
  font-size: 13px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}
.empty {
  color: var(--text-2);
  font-size: 14px;
  padding: 24px 4px;
}

@media (max-width: 760px) {
  .body {
    flex-direction: column;
  }
  .sidebar {
    width: 100%;
    max-height: 38vh;
    border-right: none;
    border-bottom: 1px solid var(--border);
  }
  .content {
    padding: 20px 14px;
  }
  .panel-head h1 {
    font-size: 22px;
  }
  .panel-head p {
    margin-bottom: 16px;
  }
  .result-head {
    flex-wrap: wrap;
  }
}
@media (max-width: 640px) {
  .pill {
    display: none;
  }
}
</style>
