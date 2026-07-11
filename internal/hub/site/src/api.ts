const BASE = '/api/mitm'

interface Flow {
  id: string
  method: string
  host: string
  path: string
  status_code: number
  duration_ms: number
  req_size: number
  resp_size: number
  captured_at: string
}

interface Rule {
  id: string
  node: string
  priority: number
  action: string
  enabled: boolean
  match: string
  spec: string
}

interface Node {
  id: string
  name: string
  token: string
  fingerprint: string
  status: string
  last_seen: string
  version: string
}

async function api<T>(method: string, path: string, body?: unknown): Promise<T> {
  const r = await fetch(BASE + path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!r.ok) { const e = await r.json(); throw new Error(e.message || r.status) }
  return r.json()
}

export function listNodes() { return api<Node[]>('GET', '/nodes') }
export function listRules() { return api<Rule[]>('GET', '/rules') }
export function listFlows() { return api<Flow[]>('GET', '/flows') }
export function createRule(r: Partial<Rule>) { return api<Rule>('POST', '/rules', r) }
export type { Flow, Rule, Node }
