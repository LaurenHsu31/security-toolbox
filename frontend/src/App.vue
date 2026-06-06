<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { listTools, runTool, type ToolMeta } from './api'
import { toolUI } from './toolsMeta'
import { loadFavorites, saveFavorites, toggleFavorite, moveFavorite, reconcile } from './prefs'
import ResultView from './components/ResultView.vue'

const tools = ref<ToolMeta[]>([])
const activeName = ref<string>('')
const inputValue = ref<string>('')
const controlValues = reactive<Record<string, string>>({})
const result = ref<unknown>(null)
const errorMsg = ref<string>('')
const loading = ref<boolean>(false)
const loadError = ref<string>('')
const favorites = ref<string[]>(loadFavorites())
const dragIndex = ref<number | null>(null)

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

const monoOutput = computed(() => {
  const key = ui.value?.monoOutputKey
  if (key && result.value && typeof result.value === 'object') {
    const v = (result.value as Record<string, unknown>)[key]
    if (typeof v === 'string') return v
  }
  return null
})

function selectTool(name: string) {
  activeName.value = name
  inputValue.value = ''
  result.value = null
  errorMsg.value = ''
  for (const k of Object.keys(controlValues)) delete controlValues[k]
  const spec = toolUI[name]
  spec?.controls?.forEach((c) => {
    controlValues[c.key] = c.default ?? ''
  })
}

function fillSample() {
  if (ui.value?.sample) inputValue.value = ui.value.sample
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

async function copyAll() {
  const text = monoOutput.value ?? JSON.stringify(result.value, null, 2)
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    /* ignore */
  }
}

onMounted(async () => {
  try {
    tools.value = await listTools()
    favorites.value = reconcile(favorites.value, tools.value.map((t) => t.name))
    if (tools.value.length) selectTool(tools.value[0].name)
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : String(e)
  }
})
</script>

<template>
  <div class="shell">
    <header class="topbar">
      <div class="brand">
        <span class="logo">◍</span>
        <span class="name">security-toolbox</span>
      </div>
      <span class="pill" title="The page is locked to its own origin by Content-Security-Policy.">
        Runs entirely on your machine · nothing is uploaded
      </span>
    </header>

    <div class="body">
      <aside class="sidebar">
        <div v-if="loadError" class="load-error">
          Could not reach the backend.<br /><small>{{ loadError }}</small>
        </div>
        <nav v-if="favoriteTools.length" class="group">
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

        <nav v-for="group in grouped" :key="group.category" class="group">
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
      </aside>

      <main class="content">
        <section v-if="activeTool" class="panel">
          <div class="panel-head">
            <h1>{{ activeTool.title }}</h1>
            <p>{{ activeTool.description }}</p>
          </div>

          <div class="card input-card">
            <div class="label-row">
              <label>{{ ui?.inputLabel }}</label>
              <button v-if="ui?.sample" class="ghost" @click="fillSample">Use sample</button>
            </div>
            <textarea
              v-model="inputValue"
              :placeholder="ui?.placeholder"
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
            <button
              v-if="result && !errorMsg"
              class="ghost"
              @click="copyAll"
            >Copy all</button>
          </div>

          <div v-if="errorMsg" class="card error">{{ errorMsg }}</div>

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
}
.logo {
  color: var(--accent);
  font-size: 18px;
}
.pill {
  font-size: 12px;
  color: var(--text-2);
  background: var(--surface-2);
  border: 1px solid var(--border);
  padding: 5px 12px;
  border-radius: 980px;
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
.load-error {
  font-size: 12px;
  color: #d70015;
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
.result-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.result-title {
  font-size: 18px;
  font-weight: 600;
}
.status {
  font-size: 13px;
  color: var(--text-2);
}
.ghost {
  margin-left: auto;
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
.label-row .ghost {
  margin-left: 0;
}
.error {
  color: #d70015;
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
</style>
