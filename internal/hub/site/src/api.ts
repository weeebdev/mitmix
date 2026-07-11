const BASE = '/api/mitm'

function getToken(): string | null {
  return localStorage.getItem('pb_token')
}

function setToken(t: string) { localStorage.setItem('pb_token', t) }
function clearToken() { localStorage.removeItem('pb_token') }

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
  const h: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) h['Authorization'] = 'Bearer ' + token
  const r = await fetch(BASE + path, { method, headers: h, body: body ? JSON.stringify(body) : undefined })
  if (r.status === 401) { clearToken(); throw new Error('unauthorized') }
  if (!r.ok) { let e; try { e = await r.json() } catch { e = { message: r.status } }; throw new Error(e.message || r.status) }
  return r.json()
}

export async function login(identity: string, password: string): Promise<void> {
  const r = await fetch('/api/collections/_superusers/auth-with-password', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ identity, password }),
  })
  if (!r.ok) throw new Error('login failed')
  const data = await r.json()
  setToken(data.token)
}

export function logout() { clearToken() }

export function isAuthed(): boolean { return !!getToken() }

export function listNodes() { return api<Node[]>('GET', '/nodes') }
export function listRules() { return api<Rule[]>('GET', '/rules') }
export function listFlows() { return api<Flow[]>('GET', '/flows') }
export function createRule(r: Partial<Rule>) { return api<Rule>('POST', '/rules', r) }
export type { Flow, Rule, Node }
