const BASE = '/api/mitm'

function getToken(): string | null {
  return localStorage.getItem('pb_token')
}

function setToken(t: string) { localStorage.setItem('pb_token', t) }
function clearToken() { localStorage.removeItem('pb_token') }

export function fmtDate(s: string | undefined | null): string {
  if (!s) return '-'
  const n = Date.parse(s)
  if (isNaN(n)) return '-'
  return new Date(n).toLocaleString()
}

export function fmtTime(s: string | undefined | null): string {
  if (!s) return '-'
  const n = Date.parse(s)
  if (isNaN(n)) return '-'
  return new Date(n).toLocaleTimeString()
}

interface Flow {
  id: string
  node: string
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
export function listFlows(params?: { host?: string; method?: string; status?: string }) {
  let q = ''
  if (params) {
    const p: string[] = []
    if (params.host) p.push('host=' + encodeURIComponent(params.host))
    if (params.method) p.push('method=' + encodeURIComponent(params.method))
    if (params.status) p.push('status=' + encodeURIComponent(params.status))
    if (p.length) q = '?' + p.join('&')
  }
  return api<Flow[]>('GET', '/flows' + q)
}
export function listFlowsByNode(node: string) { return api<Flow[]>('GET', '/flows?node=' + encodeURIComponent(node)) }
export function getFlowDetail(id: string) { return api<FlowDetail>('GET', '/flows/' + id) }
export function createRule(r: Partial<Rule>) { return api<Rule>('POST', '/rules', r) }
export function updateRule(id: string, data: Partial<Rule>) { return api<Rule>('PUT', '/rules/' + id, data) }
export function deleteRule(id: string) { return api<{deleted: string}>('DELETE', '/rules/' + id) }
export function reorderRules(ids: string[]) { return api<{ok: boolean}>('POST', '/rules/reorder', { ids }) }

interface FlowDetail {
  id: string
  node: string
  captured_at: string
  method: string
  host: string
  path: string
  status_code: number
  req_size: number
  resp_size: number
  duration_ms: number
  tags: string[]
  req_headers: Record<string, string[]>
  resp_headers: Record<string, string[]>
  req_body: string
  resp_body: string
}

interface Token {
  id: string
  token: string
  label: string
}

export function listTokens() { return api<Token[]>('GET', '/tokens') }
export function generateToken(label: string) { return api<Token>('POST', '/tokens', { label }) }
export function deleteToken(id: string) { return api<{deleted: string}>('DELETE', '/tokens/' + id) }

export function listQueries() { return api<Query[]>('GET', '/queries') }
export function createQuery(name: string, filter: string) { return api<Query>('POST', '/queries', { name, filter }) }
export function deleteQuery(id: string) { return api<{deleted: string}>('DELETE', '/queries/' + id) }
export function runQuery(id: string) { return api<Flow[]>('GET', '/queries/' + id + '/run') }

interface Query {
  id: string
  name: string
  filter: string
}

export function connectLive(cb: (msg: any) => void): () => void {
  const token = getToken()
  if (!token) return () => {}
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const ws = new WebSocket(proto + '//' + location.host + '/ws/dash')
  const sendAuth = () => {
    if (ws.readyState === WebSocket.OPEN)
      ws.send(JSON.stringify({ action: 'auth', token }))
  }
  ws.onopen = sendAuth
  ws.onmessage = (e) => {
    try { cb(JSON.parse(e.data)) } catch {}
  }
  return () => ws.close()
}

export type { Flow, FlowDetail, Rule, Node, Token, Query }
