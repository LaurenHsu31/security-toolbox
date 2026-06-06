export interface ToolMeta {
  name: string
  title: string
  category: string
  description: string
}

export async function listTools(): Promise<ToolMeta[]> {
  const res = await fetch('/api/v1/tools')
  if (!res.ok) throw new Error('could not load tools')
  return res.json()
}

export async function runTool(
  name: string,
  body: Record<string, unknown>
): Promise<unknown> {
  const res = await fetch(`/api/v1/run/${name}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  })
  let json: any
  try {
    json = await res.json()
  } catch {
    throw new Error(`server returned ${res.status}`)
  }
  if (!res.ok || json?.ok === false) {
    throw new Error(json?.error || `request failed (${res.status})`)
  }
  return json.data
}
